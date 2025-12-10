package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/franzego/stage08/config"
	"github.com/franzego/stage08/internal/repository"
	"github.com/franzego/stage08/internal/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthHandler struct {
	userRepo      *repository.UserRepository
	oauthConfig   *oauth2.Config
	jwtSecret     string
	jwtExpiration time.Duration
}

func NewAuthHandler(userRepo *repository.UserRepository, cfg *config.Config) *AuthHandler {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.Google.ClientID,
		ClientSecret: cfg.Google.ClientSecret,
		RedirectURL:  cfg.Google.RedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &AuthHandler{
		userRepo:      userRepo,
		oauthConfig:   oauthConfig,
		jwtSecret:     cfg.JWT.Secret,
		jwtExpiration: cfg.JWT.Expiration,
	}
}

// GoogleLogin initiates the Google OAuth flow
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	// Generate a random state for CSRF protection
	state := utils.GenerateRandomString(32)

	// Store state in session or cookie (simplified here)
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)

	log.Printf("Generated state: %s", state)

	url := h.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback handles the OAuth callback from Google
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	// Log all query parameters
	log.Printf("All query params: %v", c.Request.URL.Query())
	
	// Verify state
	state := c.Query("state")
	savedState, err := c.Cookie("oauth_state")

	log.Printf("Received state: '%s'", state)
	log.Printf("Saved state: '%s'", savedState)
	log.Printf("Cookie error: %v", err)

	if err != nil || state == "" || state != savedState {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid state parameter",
			"debug": gin.H{
				"received_state": state,
				"saved_state": savedState,
				"cookie_error":   fmt.Sprintf("%v", err),
				"all_params": c.Request.URL.Query(),
			},
		})
		return
	}

	// Exchange code for token
	code := c.Query("code")
	token, err := h.oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("Failed to exchange token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	// Get user info from Google
	userInfo, err := h.getUserInfo(token.AccessToken)
	if err != nil {
		log.Printf("Failed to get user info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}

	// Find or create user
	user, err := h.userRepo.FindByGoogleID(userInfo.ID)
	if err != nil {
		log.Printf("Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if user == nil {
		// Create new user
		picture := &userInfo.Picture
		if userInfo.Picture == "" {
			picture = nil
		}

		user, err = h.userRepo.Create(userInfo.ID, userInfo.Email, userInfo.Name, picture)
		if err != nil {
			log.Printf("Failed to create user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
		log.Printf("✅ New user created: %s", user.Email)
	}

	// Generate JWT
	jwtToken, err := utils.GenerateJWT(user.ID, user.Email, user.Name, h.jwtSecret, h.jwtExpiration)
	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": jwtToken,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
	})
}

// GoogleUserInfo represents the user info from Google
type GoogleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func (h *AuthHandler) getUserInfo(accessToken string) (*GoogleUserInfo, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}
