/*
 * @Author: Vincent Yang
 * @Date: 2025-04-08 22:44:31
 * @LastEditors: Vincent Yang
 * @LastEditTime: 2025-04-26 17:21:21
 * @FilePath: /raycast2api/service/handlers.go
 * @Telegram: https://t.me/missuo
 * @GitHub: https://github.com/missuo
 *
 * Copyright © 2025 by Vincent, All Rights Reserved.
 */

package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// handleChatCompletions handles OpenAI chat completions endpoint
func handleChatCompletions(c *gin.Context, config Config) {
	var body OpenAIChatRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Details string `json:"details,omitempty"`
			}{
				Message: "Invalid request body",
				Type:    "invalid_request_error",
				Details: err.Error(),
			},
		})
		return
	}

	if len(body.Messages) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Details string `json:"details,omitempty"`
			}{
				Message: "Missing or invalid 'messages' field",
				Type:    "invalid_request_error",
			},
		})
		return
	}

	// Use default model if not specified
	model := body.Model
	if model == "" {
		model = DefaultModel
	}

	// Use default temperature if not specified
	temperature := body.Temperature
	if temperature == 0 {
		temperature = 0.5
	}

	stream := body.Stream

	// Get models from cache or fetch them if cache is expired
	models, err := config.ModelCache.GetModels(config)
	if err != nil {
		log.Printf("Warning: Using models with possible error: %v", err)
	}

	// Get provider info from the models
	provider, modelName := getProviderInfo(model, models)
	log.Printf("Using provider: %s, model: %s", provider, modelName)

	// Create a unique thread ID for this conversation
	threadId := uuid.New().String()

	// Convert messages and extract system instruction
	messageResult := convertMessages(body.Messages)

	// Prepare Raycast request
	raycastRequest := RaycastChatRequest{
		AdditionalSystemInstructions: "",
		Debug:                        false,
		Locale:                       "en-US",
		Messages:                     messageResult.RaycastMessages,
		Model:                        modelName,
		Provider:                     provider,
		Source:                       "ai_chat",
		SystemInstruction:            messageResult.SystemInstruction,
		Temperature:                  temperature,
		ThreadID:                     threadId,
		Tools: []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		}{
			// Uncomment to enable tools
			// {Name: "web_search", Type: "remote_tool"},
			// {Name: "search_images", Type: "remote_tool"},
		},
	}

	requestBody, err := json.Marshal(raycastRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Details string `json:"details,omitempty"`
			}{
				Message: "Failed to marshal request",
				Type:    "server_error",
				Details: err.Error(),
			},
		})
		return
	}

	log.Printf("Sending request to Raycast: %s", string(requestBody))

	client := &http.Client{
		Timeout: 5 * time.Minute, // Longer timeout for chat completions
	}
	req, err := http.NewRequest("POST", RaycastAPIURL, bytes.NewBuffer(requestBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Details string `json:"details,omitempty"`
			}{
				Message: "Error creating request",
				Type:    "server_error",
				Details: err.Error(),
			},
		})
		return
	}

	for key, value := range getRaycastHeaders(config) {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Details string `json:"details,omitempty"`
			}{
				Message: fmt.Sprintf("Error sending request to Raycast: %v", err),
				Type:    "relay_error",
				Details: err.Error(),
			},
		})
		return
	}
	defer resp.Body.Close()

	log.Printf("Response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		errorText := string(bodyBytes)

		// Try to parse error as JSON
		var errorJson map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &errorJson); err == nil {
			jsonBytes, _ := json.Marshal(errorJson)
			errorText = string(jsonBytes)
		}

		c.JSON(resp.StatusCode, ErrorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Details string `json:"details,omitempty"`
			}{
				Message: fmt.Sprintf("Raycast API error: %d %s", resp.StatusCode, errorText),
				Type:    "relay_error",
			},
		})
		return
	}

	// Handle streaming response
	if stream {
		handleStreamingResponse(c, resp, model)
	} else {
		handleNonStreamingResponse(c, resp, model)
	}
}

// handleModels handles models endpoint
func handleModels(c *gin.Context, config Config) {
	// Get models from cache or fetch them if cache is expired
	models, err := config.ModelCache.GetModels(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Details string `json:"details,omitempty"`
			}{
				Message: fmt.Sprintf("An error occurred while fetching models: %v", err),
				Type:    "relay_error",
				Details: err.Error(),
			},
		})
		return
	}

	// Convert models to a slice that can be sorted
	var modelSlice []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}

	for _, info := range models {
		modelSlice = append(modelSlice, struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		}{
			ID:      info.Model,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: info.Provider,
		})
	}

	// Sort by ID
	sort.Slice(modelSlice, func(i, j int) bool {
		return modelSlice[i].ID < modelSlice[j].ID
	})

	// Create OpenAI format response
	openaiModels := OpenAIModelResponse{
		Object: "list",
		Data:   modelSlice,
	}

	jsonData, err := json.MarshalIndent(openaiModels, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Details string `json:"details,omitempty"`
			}{
				Message: "Error formatting JSON response",
				Type:    "server_error",
				Details: err.Error(),
			},
		})
		return
	}

	// Add newline to the end of JSON data
	jsonData = append(jsonData, '\n')

	// Set content type and write formatted JSON
	c.Header("Content-Type", "application/json")
	c.Writer.Write(jsonData)
}

// handleRefreshModels handles manual refresh of the model cache
func handleRefreshModels(c *gin.Context, config Config) {
	config.ModelCache.ForceCacheRefresh(config)
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Model cache refreshed",
	})
}
