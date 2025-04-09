/*
 * @Author: Vincent Yang
 * @Date: 2025-04-04 16:14:09
 * @LastEditors: Vincent Yang
 * @LastEditTime: 2025-04-08 22:46:07
 * @FilePath: /raycast2api/main.go
 * @Telegram: https://t.me/missuo
 * @GitHub: https://github.com/missuo
 *
 * Copyright © 2025 by Vincent, All Rights Reserved.
 */

package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// Main function
func main() {
	// Configure logging
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Initialize model cache
	modelCache := NewModelCache()

	// Load configuration from environment variables
	config := Config{
		RaycastBearerToken: os.Getenv("RAYCAST_BEARER_TOKEN"),
		APIKey:             os.Getenv("API_KEY"),
		ModelCache:         modelCache,
	}

	// Log environment variables status
	log.Printf("RAYCAST_BEARER_TOKEN: %s", map[bool]string{true: "Set", false: "Not set"}[config.RaycastBearerToken != ""])
	log.Printf("API_KEY: %s", map[bool]string{true: "Set", false: "Not set"}[config.APIKey != ""])

	// Validate required environment variables
	if config.RaycastBearerToken == "" {
		log.Fatal("Missing required environment variable: RAYCAST_BEARER_TOKEN")
	}

	// Set Release Mode
	gin.SetMode(gin.ReleaseMode)

	// Initialize Gin router
	router := gin.Default()

	// Setup middlewares
	setupMiddlewares(router, config)

	// Setup routes
	setupRoutes(router, config)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// setupMiddlewares configures all middlewares for the router
func setupMiddlewares(router *gin.Engine, config Config) {
	// Handle CORS preflight requests
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	})

	// API key validation middleware
	router.Use(func(c *gin.Context) {
		if !validateAPIKey(c, config) {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: struct {
					Message string `json:"message"`
					Type    string `json:"type"`
					Details string `json:"details,omitempty"`
				}{
					Message: "Invalid API key",
					Type:    "authentication_error",
				},
			})
			c.Abort()
			return
		}
		c.Next()
	})

	// Log request middleware
	router.Use(func(c *gin.Context) {
		timestamp := time.Now().Format(time.RFC3339)
		log.Printf("[%s] %s %s", timestamp, c.Request.Method, c.Request.URL.Path)
		c.Next()
	})
}

// setupRoutes configures all routes for the application
func setupRoutes(router *gin.Engine, config Config) {
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		handleChatCompletions(c, config)
	})

	router.GET("/v1/models", func(c *gin.Context) {
		handleModels(c, config)
	})

	router.GET("/v1/refresh-models", func(c *gin.Context) {
		handleRefreshModels(c, config)
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
