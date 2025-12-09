package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/franzego/stage08/config"
	"github.com/franzego/stage08/internal/middleware"
	"github.com/franzego/stage08/internal/models"
	"github.com/franzego/stage08/internal/paystack"
	"github.com/franzego/stage08/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PaystackHandler struct {
	paystackClient *paystack.Client
	walletRepo     *repository.WalletRepository
	txRepo         *repository.TransactionRepository
	db             *sqlx.DB
}

func NewPaystackHandler(cfg *config.PaystackConfig, walletRepo *repository.WalletRepository, txRepo *repository.TransactionRepository, db *sqlx.DB) *PaystackHandler {
	return &PaystackHandler{
		paystackClient: paystack.NewClient(cfg.SecretKey),
		walletRepo:     walletRepo,
		txRepo:         txRepo,
		db:             db,
	}
}

// InitializeDeposit initializes a Paystack deposit
// POST /wallet/deposit
func (h *PaystackHandler) InitializeDeposit(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Amount int64 `json:"amount" binding:"required,min=100"` // Minimum 100 kobo (1 Naira)
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request. Amount must be at least 100 kobo"})
		return
	}

	// Get user's wallet and email
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

	// Get user email (we need it from context or fetch user)
	email := middleware.GetUserEmail(c)

	// Generate unique reference
	reference := fmt.Sprintf("DEP_%s_%s", userID.String()[:8], uuid.New().String()[:8])

	// Create pending transaction
	tx := &models.Transaction{
		UserID:      userID,
		WalletID:    wallet.ID,
		Type:        models.TransactionTypeDeposit,
		Amount:      req.Amount,
		Status:      models.TransactionStatusPending,
		Reference:   &reference,
		Description: stringPtr("Wallet deposit via Paystack"),
	}

	if err := h.txRepo.Create(tx); err != nil {
		log.Printf("Failed to create transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction"})
		return
	}

	// Initialize Paystack transaction
	paystackResp, err := h.paystackClient.InitializeTransaction(email, req.Amount, reference)
	if err != nil {
		log.Printf("Paystack initialization failed: %v", err)
		// Update transaction status to failed
		h.txRepo.UpdateStatus(tx.ID, models.TransactionStatusFailed)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reference":         reference,
		"authorization_url": paystackResp.Data.AuthorizationURL,
	})
}

// PaystackWebhook handles Paystack webhook notifications
// POST /wallet/paystack/webhook
func (h *PaystackHandler) PaystackWebhook(c *gin.Context) {
	// Read raw body for signature verification
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Failed to read webhook body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request"})
		return
	}

	// Verify signature
	signature := c.GetHeader("x-paystack-signature")
	if signature == "" {
		log.Println("Missing Paystack signature")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing signature"})
		return
	}

	if !h.paystackClient.VerifyWebhookSignature(signature, body) {
		log.Println("Invalid Paystack signature")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	// Parse webhook event
	var event paystack.WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("Failed to parse webhook: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	// Only process successful charge events
	if event.Event != "charge.success" {
		c.JSON(http.StatusOK, gin.H{"status": true})
		return
	}

	// Process the deposit (idempotent)
	if err := h.processDeposit(event.Data.Reference, event.Data.Amount, event.Data.Status); err != nil {
		log.Printf("Failed to process deposit: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process deposit"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": true})
}

// GetDepositStatus checks the status of a deposit
// GET /wallet/deposit/:reference/status
func (h *PaystackHandler) GetDepositStatus(c *gin.Context) {
	reference := c.Param("reference")

	// Find transaction
	tx, err := h.txRepo.FindByReference(reference)
	if err != nil {
		log.Printf("Failed to find transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if tx == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reference": reference,
		"status":    tx.Status,
		"amount":    tx.Amount,
	})
}

// processDeposit credits wallet after successful payment (idempotent)
func (h *PaystackHandler) processDeposit(reference string, amount int64, status string) error {
	// Find transaction by reference
	tx, err := h.txRepo.FindByReference(reference)
	if err != nil {
		return fmt.Errorf("failed to find transaction: %w", err)
	}

	if tx == nil {
		return fmt.Errorf("transaction not found: %s", reference)
	}

	// Check if already processed (idempotency)
	if tx.Status == models.TransactionStatusSuccess {
		log.Printf("Transaction %s already processed, skipping", reference)
		return nil
	}

	// Verify status
	if status != "success" {
		// Update to failed
		return h.txRepo.UpdateStatus(tx.ID, models.TransactionStatusFailed)
	}

	// Verify amount matches
	if tx.Amount != amount {
		log.Printf("Amount mismatch for %s: expected %d, got %d", reference, tx.Amount, amount)
		return h.txRepo.UpdateStatus(tx.ID, models.TransactionStatusFailed)
	}

	// Begin database transaction for atomic operation
	dbTx, err := h.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer dbTx.Rollback()

	// Credit wallet
	query := `UPDATE wallets SET balance = balance + $1, updated_at = NOW() WHERE id = $2`
	if _, err := dbTx.Exec(query, amount, tx.WalletID); err != nil {
		return fmt.Errorf("failed to credit wallet: %w", err)
	}

	// Update transaction status
	updateQuery := `UPDATE transactions SET status = $1, updated_at = NOW() WHERE id = $2`
	if _, err := dbTx.Exec(updateQuery, models.TransactionStatusSuccess, tx.ID); err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	// Commit transaction
	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("✅ Deposit processed: %s, amount: %d kobo", reference, amount)
	return nil
}

func stringPtr(s string) *string {
	return &s
}
