package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Standalone JWT generator - no dependencies on main app
func main() {
	// Test user details
	userID := uuid.New()
	email := "test@example.com"
	name := "Test User"
	secret := "super-secret-jwt-key-change-in-production" // Must match .env
	expiration := 24 * time.Hour

	// Generate JWT
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"email":   email,
		"name":    name,
		"exp":     time.Now().Add(expiration).Unix(),
		"iat":     time.Now().Unix(),
		"nbf":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		fmt.Printf("Error generating token: %v\n", err)
		return
	}

	fmt.Println("========================================")
	fmt.Println("Test JWT Token Generated!")
	fmt.Println("========================================")
	fmt.Printf("User ID: %s\n", userID)
	fmt.Printf("Email: %s\n", email)
	fmt.Printf("Name: %s\n", name)
	fmt.Println("----------------------------------------")
	fmt.Println("Token:")
	fmt.Println(tokenString)
	fmt.Println("----------------------------------------")
	fmt.Println("\nTest with curl:")
	fmt.Printf("curl -H \"Authorization: Bearer %s\" http://localhost:8080/me\n", tokenString)
	fmt.Println("========================================")
}
