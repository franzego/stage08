package handlers

import (
	"log"
	"net/http"

	"github.com/franzego/stage08/internal/middleware"
	"github.com/franzego/stage08/internal/repository"
	"github.com/franzego/stage08/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type APIKeyHandler struct {
	apiKeyRepo *repository.APIKeyRepository
}

func NewAPIKeyHandler(apiKeyRepo *repository.APIKeyRepository) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyRepo: apiKeyRepo,
	}
}

// CreateAPIKey creates a new API key for the user
// POST /keys/create
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Name        string   `json:"name" binding:"required"`
		Permissions []string `json:"permissions" binding:"required"`
		Expiry      string   `json:"expiry" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate permissions
	if err := utils.ValidatePermissions(req.Permissions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse expiry
	expiresAt, err := utils.ParseExpiry(req.Expiry)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user has reached the limit of 5 active keys
	count, err := h.apiKeyRepo.CountActiveByUser(userID)
	if err != nil {
		log.Printf("Failed to count API keys: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if count >= 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 5 active API keys allowed per user"})
		return
	}

	// Create API key
	apiKey, rawKey, err := h.apiKeyRepo.Create(userID, req.Name, req.Permissions, expiresAt)
	if err != nil {
		log.Printf("Failed to create API key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"api_key":    rawKey,
		"expires_at": apiKey.ExpiresAt,
	})
}

// RolloverAPIKey creates a new API key with the same permissions as an expired key
// POST /keys/rollover
func (h *APIKeyHandler) RolloverAPIKey(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		ExpiredKeyID string `json:"expired_key_id" binding:"required"`
		Expiry       string `json:"expiry" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Parse expired key ID
	expiredKeyID, err := uuid.Parse(req.ExpiredKeyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expired_key_id"})
		return
	}

	// Find the expired key
	expiredKey, err := h.apiKeyRepo.FindByID(expiredKeyID)
	if err != nil {
		log.Printf("Failed to find API key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if expiredKey == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	// Verify ownership
	if expiredKey.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not own this API key"})
		return
	}

	// Verify it's actually expired
	if !expiredKey.IsExpired() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API key is not expired yet"})
		return
	}

	// Parse new expiry
	expiresAt, err := utils.ParseExpiry(req.Expiry)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check active key limit
	count, err := h.apiKeyRepo.CountActiveByUser(userID)
	if err != nil {
		log.Printf("Failed to count API keys: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if count >= 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 5 active API keys allowed per user"})
		return
	}

	// Create new API key with same permissions
	apiKey, rawKey, err := h.apiKeyRepo.Create(userID, expiredKey.Name, expiredKey.Permissions, expiresAt)
	if err != nil {
		log.Printf("Failed to create API key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"api_key":    rawKey,
		"expires_at": apiKey.ExpiresAt,
	})
}

// ListAPIKeys lists all API keys for the user (without revealing the actual keys)
// GET /keys/list
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	keys, err := h.apiKeyRepo.ListByUser(userID)
	if err != nil {
		log.Printf("Failed to list API keys: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Don't expose key hashes
	response := make([]gin.H, len(keys))
	for i, key := range keys {
		response[i] = gin.H{
			"id":          key.ID,
			"name":        key.Name,
			"key_prefix":  key.KeyPrefix,
			"permissions": key.Permissions,
			"is_active":   key.IsActive,
			"expires_at":  key.ExpiresAt,
			"last_used":   key.LastUsedAt,
			"created_at":  key.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{"keys": response})
}

// RevokeAPIKey deactivates an API key
// POST /keys/revoke
func (h *APIKeyHandler) RevokeAPIKey(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		KeyID string `json:"key_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	keyID, err := uuid.Parse(req.KeyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid key_id"})
		return
	}

	// Find the key
	key, err := h.apiKeyRepo.FindByID(keyID)
	if err != nil {
		log.Printf("Failed to find API key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if key == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	// Verify ownership
	if key.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not own this API key"})
		return
	}

	// Revoke the key
	if err := h.apiKeyRepo.Revoke(keyID); err != nil {
		log.Printf("Failed to revoke API key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked successfully"})
}
