package main

import (
	"log"
	"os"

	"github.com/franzego/stage08/config"
	"github.com/franzego/stage08/internal/database"
	"github.com/franzego/stage08/internal/handlers"
	"github.com/franzego/stage08/internal/middleware"
	"github.com/franzego/stage08/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Connect to database
	db, err := database.Connect(&cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	walletRepo := repository.NewWalletRepository(db)
	txRepo := repository.NewTransactionRepository(db)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, cfg)
	apiKeyHandler := handlers.NewAPIKeyHandler(apiKeyRepo)
	walletHandler := handlers.NewWalletHandler(walletRepo, txRepo, db)
	paystackHandler := handlers.NewPaystackHandler(&cfg.Paystack, walletRepo, txRepo, db)

	// Initialize Gin router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Wallet service is running",
		})
	})

	// Swagger documentation endpoint
	router.GET("/swagger.yaml", func(c *gin.Context) {
		data, err := os.ReadFile("swagger.yaml")
		if err != nil {
			c.JSON(404, gin.H{"error": "Swagger file not found"})
			return
		}
		c.Data(200, "application/x-yaml", data)
	})

	// Auth routes (no authentication required)
	authGroup := router.Group("/auth")
	{
		authGroup.GET("/google", authHandler.GoogleLogin)
		authGroup.GET("/google/callback", authHandler.GoogleCallback)
	}

	// API Key routes (JWT required)
	keysGroup := router.Group("/keys")
	keysGroup.Use(middleware.JWTAuth(cfg.JWT.Secret))
	{
		keysGroup.POST("/create", apiKeyHandler.CreateAPIKey)
		keysGroup.POST("/rollover", apiKeyHandler.RolloverAPIKey)
		keysGroup.GET("/list", apiKeyHandler.ListAPIKeys)
		keysGroup.POST("/revoke", apiKeyHandler.RevokeAPIKey)
	}

	// Wallet routes (JWT or API key required)
	walletGroup := router.Group("/wallet")
	walletGroup.Use(middleware.AuthMiddleware(cfg.JWT.Secret, apiKeyRepo))
	{
		// Balance endpoint - requires 'read' permission
		walletGroup.GET("/balance",
			middleware.RequirePermission("read"),
			walletHandler.GetBalance,
		)

		// Transaction history - requires 'read' permission
		walletGroup.GET("/transactions",
			middleware.RequirePermission("read"),
			walletHandler.GetTransactions,
		)

		// Deposit endpoint - requires 'deposit' permission
		walletGroup.POST("/deposit",
			middleware.RequirePermission("deposit"),
			paystackHandler.InitializeDeposit,
		)

		// Transfer endpoint - requires 'transfer' permission
		walletGroup.POST("/transfer",
			middleware.RequirePermission("transfer"),
			walletHandler.Transfer,
		)

		// Deposit status check - requires 'read' permission
		walletGroup.GET("/deposit/:reference/status",
			middleware.RequirePermission("read"),
			paystackHandler.GetDepositStatus,
		)
	}

	// Paystack webhook (no authentication - validated by signature)
	router.POST("/wallet/paystack/webhook", paystackHandler.PaystackWebhook)

	// Protected routes (JWT required) - for testing
	protectedGroup := router.Group("/")
	protectedGroup.Use(middleware.JWTAuth(cfg.JWT.Secret))
	{
		// Test protected endpoint
		protectedGroup.GET("/me", func(c *gin.Context) {
			userID, _ := middleware.GetUserID(c)
			email := middleware.GetUserEmail(c)

			c.JSON(200, gin.H{
				"user_id": userID,
				"email":   email,
			})
		})
	}

	// Start server
	log.Printf("Server starting on port %s...", cfg.Server.Port)
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
