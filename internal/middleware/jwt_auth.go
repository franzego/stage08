package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/franzego/stage08/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// JWTAuth middleware validates JWT tokens
func JWTAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
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

		// Validate token
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

		c.Next()
	}
}

// GetUserID retrieves the user ID from context
func GetUserID(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("user_id not found in context")
	}

	uid, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("user_id is not a valid UUID")
	}

	return uid, nil
}

// GetUserEmail retrieves the user email from context
func GetUserEmail(c *gin.Context) string {
	email, _ := c.Get("user_email")
	return email.(string)
}
