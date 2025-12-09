package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/franzego/stage08/internal/repository"
	"github.com/franzego/stage08/internal/utils"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware handles both JWT and API key authentication
func AuthMiddleware(jwtSecret string, apiKeyRepo *repository.APIKeyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for API key first (x-api-key header)
		apiKey := c.GetHeader("x-api-key")
		if apiKey != "" {
			if err := validateAPIKey(c, apiKey, apiKeyRepo); err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// Fall back to JWT authentication
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header or x-api-key required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate JWT
		claims, err := utils.ValidateJWT(token, jwtSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_name", claims.Name)
		c.Set("auth_type", "jwt")
		c.Set("permissions", []string{"deposit", "transfer", "read"}) // JWT has all permissions

		c.Next()
	}
}

// validateAPIKey validates an API key and sets user context
func validateAPIKey(c *gin.Context, rawKey string, apiKeyRepo *repository.APIKeyRepository) error {
	// Find the API key
	apiKey, err := apiKeyRepo.FindByKey(rawKey)
	if err != nil {
		log.Printf("Failed to find API key: %v", err)
		return err
	}

	if apiKey == nil {
		return fmt.Errorf("invalid API key")
	}

	// Check if active
	if !apiKey.IsActive {
		return fmt.Errorf("API key is revoked")
	}

	// Check if expired
	if apiKey.IsExpired() {
		return fmt.Errorf("API key has expired")
	}

	// Update last used timestamp (async)
	go apiKeyRepo.UpdateLastUsed(apiKey.ID)

	// Store user info and permissions in context
	c.Set("user_id", apiKey.UserID)
	c.Set("auth_type", "apikey")
	c.Set("permissions", apiKey.Permissions)
	c.Set("api_key_id", apiKey.ID)

	return nil
}

// RequirePermission middleware checks if the user has a specific permission
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		permissions, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "No permissions found"})
			c.Abort()
			return
		}

		perms, ok := permissions.([]string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid permissions format"})
			c.Abort()
			return
		}

		// Check if permission exists
		hasPermission := false
		for _, p := range perms {
			if p == permission {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("Permission '%s' required", permission),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
