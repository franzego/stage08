package main

import (
	"log"

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

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, cfg)

	// Initialize Gin router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Wallet service is running",
		})
	})

	// Auth routes (no authentication required)
	authGroup := router.Group("/auth")
	{
		authGroup.GET("/google", authHandler.GoogleLogin)
		authGroup.GET("/google/callback", authHandler.GoogleCallback)
	}

	// Protected routes (JWT required)
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
