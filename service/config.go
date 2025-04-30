/*
 * @Author: Vincent Yang
 * @Date: 2025-04-08 22:43:16
 * @LastEditors: Vincent Yang
 * @LastEditTime: 2025-04-29 20:56:57
 * @FilePath: /raycast2api/service/config.go
 * @Telegram: https://t.me/missuo
 * @GitHub: https://github.com/missuo
 *
 * Copyright © 2025 by Vincent, All Rights Reserved.
 */

package service

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Configuration constants
const (
	RaycastAPIURL    = "https://backend.raycast.com/api/v1/ai/chat_completions"
	RaycastModelsURL = "https://backend.raycast.com/api/v1/ai/models"
	UserAgent        = "Raycast/1.96.3 (macOS Version 15.5 (Build 24F5068b))"
	DefaultProvider  = "anthropic"
	DefaultModel     = "claude-3-7-sonnet-latest"
	ModelCacheTTL    = 6 * time.Hour // Cache models for 6 hours
)

// Config represents the application configuration
type Config struct {
	RaycastBearerToken string
	APIKey             string
	ModelCache         *ModelCache
	Port               string
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Details string `json:"details,omitempty"`
	} `json:"error"`
}

// ModelCache represents the cache for models
type ModelCache struct {
	models    map[string]ModelCacheEntry
	expiresAt time.Time
	mutex     sync.RWMutex
}

// ModelCacheEntry stores information about a model
type ModelCacheEntry struct {
	Model    string `json:"model"`
	Provider string `json:"provider"`
}

// validateAPIKey validates the API key from the request
func validateAPIKey(c *gin.Context, config Config) bool {
	if config.APIKey == "" {
		return true // If no API key is set, allow all requests
	}

	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false
	}

	// Extract the token from the Authorization header
	token := strings.TrimPrefix(authHeader, "Bearer ")

	// Split the config.APIKey by comma and trim spaces
	validKeys := strings.Split(config.APIKey, ",")
	for _, key := range validKeys {
		if strings.TrimSpace(key) == token {
			return true
		}
	}

	return false
}

// getRaycastHeaders returns headers for Raycast API requests
func getRaycastHeaders(config Config) map[string]string {
	return map[string]string{
		"Host":            "backend.raycast.com",
		"Accept":          "application/json",
		"User-Agent":      UserAgent,
		"Authorization":   "Bearer " + config.RaycastBearerToken,
		"Accept-Language": "en-US,en;q=0.9",
		"Content-Type":    "application/json",
		"Connection":      "close",
	}
}

// InitConfig initializes the configuration
func InitConfig() *Config {
	// Initialize model cache
	modelCache := NewModelCache()

	// Load configuration from environment variables
	config := &Config{
		RaycastBearerToken: os.Getenv("RAYCAST_BEARER_TOKEN"),
		APIKey:             os.Getenv("API_KEY"),
		ModelCache:         modelCache,
		Port:               os.Getenv("PORT"),
	}

	// Log environment variable status
	log.Printf("RAYCAST_BEARER_TOKEN: %s", map[bool]string{true: "Set", false: "Not set"}[config.RaycastBearerToken != ""])
	log.Printf("API_KEY: %s", map[bool]string{true: "Set", false: "Not set"}[config.APIKey != ""])

	// Validate required environment variables
	if config.RaycastBearerToken == "" {
		log.Fatal("Missing required environment variable: RAYCAST_BEARER_TOKEN")
	}

	if config.Port == "" {
		config.Port = "8080"
	}

	return config
}
