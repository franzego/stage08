package handlers

import (
	"log"
	"net/http"

	"github.com/franzego/stage08/internal/middleware"
	"github.com/franzego/stage08/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type WalletHandler struct {
	walletRepo *repository.WalletRepository
	txRepo     *repository.TransactionRepository
	db         *sqlx.DB
}

func NewWalletHandler(walletRepo *repository.WalletRepository, txRepo *repository.TransactionRepository, db *sqlx.DB) *WalletHandler {
	return &WalletHandler{
		walletRepo: walletRepo,
		txRepo:     txRepo,
		db:         db,
	}
}

// GetBalance returns the user's wallet balance
// GET /wallet/balance
func (h *WalletHandler) GetBalance(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	wallet, err := h.walletRepo.FindByUserID(userID)
	if err != nil {
		log.Printf("Failed to find wallet: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if wallet == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"balance":       wallet.Balance,
		"wallet_number": wallet.WalletNumber,
	})
}

// GetTransactions returns the user's transaction history
// GET /wallet/transactions
func (h *WalletHandler) GetTransactions(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get pagination parameters
	limit := 50
	offset := 0

	transactions, err := h.txRepo.ListByUser(userID, limit, offset)
	if err != nil {
		log.Printf("Failed to list transactions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Format response
	response := make([]gin.H, len(transactions))
	for i, tx := range transactions {
		response[i] = gin.H{
			"type":   tx.Type,
			"amount": tx.Amount,
			"status": tx.Status,
		}
	}
	c.JSON(http.StatusOK, response)
}

// Transfer sends money from user's wallet to another wallet
// POST /wallet/transfer
func (h *WalletHandler) Transfer(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		WalletNumber string `json:"wallet_number" binding:"required"`
		Amount       int64  `json:"amount" binding:"required,min=100"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get sender wallet
	senderWallet, err := h.walletRepo.FindByUserID(userID)
	if err != nil {
		log.Printf("Failed to find sender wallet: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Get recipient wallet
	recipientWallet, err := h.walletRepo.FindByWalletNumber(req.WalletNumber)
	if err != nil {
		log.Printf("Failed to find recipient wallet: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if recipientWallet == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recipient wallet not found"})
		return
	}

	// Cannot transfer to self
	if senderWallet.ID == recipientWallet.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot transfer to yourself"})
		return
	}

	// Check balance
	if senderWallet.Balance < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient balance"})
		return
	}

	// Debit sender
	if err := h.walletRepo.Debit(senderWallet.ID, req.Amount); err != nil {
		log.Printf("Failed to debit sender: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient balance"})
		return
	}

	// Credit recipient
	if err := h.walletRepo.Credit(recipientWallet.ID, req.Amount); err != nil {
		log.Printf("Failed to credit recipient: %v", err)
		// Rollback: credit back sender
		h.walletRepo.Credit(senderWallet.ID, req.Amount)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transfer failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Transfer completed",
	})
}
