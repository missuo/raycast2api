/*
 * @Author: Vincent Yang
 * @Date: 2025-04-09 15:40:07
 * @LastEditors: Vincent Yang
 * @LastEditTime: 2025-04-09 16:16:38
 * @FilePath: /raycast2api/service/router.go
 * @Telegram: https://t.me/missuo
 * @GitHub: https://github.com/missuo
 *
 * Copyright © 2025 by Vincent, All Rights Reserved.
 */

package service

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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
func Router(config *Config) *gin.Engine {
	router := gin.Default()
	setupMiddlewares(router, *config) // Dereference when passing to setupMiddlewares
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		handleChatCompletions(c, *config) // Dereference when passing to handlers
	})

	router.GET("/v1/models", func(c *gin.Context) {
		handleModels(c, *config) // Dereference when passing to handlers
	})

	router.GET("/v1/refresh-models", func(c *gin.Context) {
		handleRefreshModels(c, *config) // Dereference when passing to handlers
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return router
}
