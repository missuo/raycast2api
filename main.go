/*
 * @Author: Vincent Yang
 * @Date: 2025-04-04 16:14:09
 * @LastEditors: Vincent Yang
 * @LastEditTime: 2025-04-05 14:41:51
 * @FilePath: /raycast2api/main.go
 * @Telegram: https://t.me/missuo
 * @GitHub: https://github.com/missuo
 *
 * Copyright © 2025 by Vincent, All Rights Reserved.
 */

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Configuration constants
const (
	RaycastAPIURL    = "https://backend.raycast.com/api/v1/ai/chat_completions"
	RaycastModelsURL = "https://backend.raycast.com/api/v1/ai/models"
	UserAgent        = "Raycast/1.94.2 (macOS Version 15.3.2 (Build 24D81))"
	DefaultProvider  = "anthropic"
	DefaultModel     = "claude-3-7-sonnet-latest"
)

// ModelCacheEntry stores information about a model
type ModelCacheEntry struct {
	Model    string `json:"model"`
	Provider string `json:"provider"`
}

// OpenAIMessage represents a message in OpenAI format
type OpenAIMessage struct {
	Role    string      `json:"role"`    // "user", "assistant", or "system"
	Content interface{} `json:"content"` // Can be string or array
}

// RaycastMessage represents a message in Raycast format
type RaycastMessage struct {
	Author  string `json:"author"` // "user" or "assistant"
	Content struct {
		Text string `json:"text"`
	} `json:"content"`
}

// RaycastChatRequest represents a chat request to Raycast API
type RaycastChatRequest struct {
	AdditionalSystemInstructions string           `json:"additional_system_instructions"`
	Debug                        bool             `json:"debug"`
	Locale                       string           `json:"locale"`
	Messages                     []RaycastMessage `json:"messages"`
	Model                        string           `json:"model"`
	Provider                     string           `json:"provider"`
	Source                       string           `json:"source"`
	SystemInstruction            string           `json:"system_instruction"`
	Temperature                  float64          `json:"temperature"`
	ThreadID                     string           `json:"thread_id"`
	Tools                        []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"tools"`
}

// OpenAIChatRequest represents a chat request in OpenAI format
type OpenAIChatRequest struct {
	Messages    []OpenAIMessage        `json:"messages"`
	Model       string                 `json:"model"`
	Temperature float64                `json:"temperature,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Extra       map[string]interface{} `json:"-"`
}

// OpenAIChatResponse represents a chat response in OpenAI format
type OpenAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role        string   `json:"role"`
			Content     string   `json:"content"`
			Refusal     *string  `json:"refusal"`
			Annotations []string `json:"annotations"`
		} `json:"message"`
		Logprobs     *string `json:"logprobs"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		TotalTokens         int `json:"total_tokens"`
		PromptTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
			AudioTokens  int `json:"audio_tokens"`
		} `json:"prompt_tokens_details"`
		CompletionTokensDetails struct {
			ReasoningTokens          int `json:"reasoning_tokens"`
			AudioTokens              int `json:"audio_tokens"`
			AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
			RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
		} `json:"completion_tokens_details"`
	} `json:"usage"`
	ServiceTier       string `json:"service_tier"`
	SystemFingerprint string `json:"system_fingerprint"`
}

// RaycastSSEData represents SSE data from Raycast
type RaycastSSEData struct {
	Text         string `json:"text,omitempty"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// OpenAIModelResponse represents a model list response in OpenAI format
type OpenAIModelResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

// Config represents the application configuration
type Config struct {
	RaycastBearerToken string
	APIKey             string
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Details string `json:"details,omitempty"`
	} `json:"error"`
}

// fetchModels fetches model information from Raycast API
func fetchModels(config Config) (map[string]ModelCacheEntry, error) {
	log.Println("Fetching models from Raycast API...")

	client := &http.Client{}
	req, err := http.NewRequest("GET", RaycastModelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	for key, value := range getRaycastHeaders(config) {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Raycast API error: %d %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if len(bodyBytes) == 0 || strings.TrimSpace(string(bodyBytes)) == "" {
		return nil, fmt.Errorf("empty response from Raycast API")
	}

	var response struct {
		Models []struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
		} `json:"models"`
	}

	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	models := make(map[string]ModelCacheEntry)
	for _, model := range response.Models {
		models[model.Model] = ModelCacheEntry{
			Provider: model.Provider,
			Model:    model.Model,
		}
	}

	log.Printf("Fetched %d models from Raycast API", len(models))
	return models, nil
}

// getRaycastHeaders returns headers for Raycast API requests
func getRaycastHeaders(config Config) map[string]string {
	return map[string]string{
		"Host":            "backend.raycast.com",
		"Accept":          "application/json",
		"User-Agent":      UserAgent,
		"Authorization":   fmt.Sprintf("Bearer %s", config.RaycastBearerToken),
		"Accept-Language": "en-US,en;q=0.9",
		"Content-Type":    "application/json",
		"Connection":      "close",
	}
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

// getProviderInfo gets provider info for a model
func getProviderInfo(modelID string, models map[string]ModelCacheEntry) (string, string) {
	if model, ok := models[modelID]; ok {
		return model.Provider, model.Model
	}
	// Fallback to defaults
	return DefaultProvider, DefaultModel
}

// convertMessages converts OpenAI messages format to Raycast format
func convertMessages(openaiMessages []OpenAIMessage) []RaycastMessage {
	raycastMessages := make([]RaycastMessage, len(openaiMessages))
	for i, msg := range openaiMessages {
		author := "user"
		if msg.Role == "assistant" {
			author = "assistant"
		}

		var contentText string
		switch content := msg.Content.(type) {
		case string:
			contentText = content
		case []interface{}:
			// Handle array content (extract text parts)
			for _, part := range content {
				if partMap, ok := part.(map[string]interface{}); ok {
					if partMap["type"] == "text" {
						if textValue, ok := partMap["text"].(string); ok {
							contentText += textValue
						}
					}
				}
			}
		}

		raycastMessages[i] = RaycastMessage{
			Author: author,
			Content: struct {
				Text string `json:"text"`
			}{
				Text: contentText,
			},
		}
	}
	return raycastMessages
}

// parseSSEResponse parses SSE response from Raycast into a single text
func parseSSEResponse(responseText string) string {
	scanner := bufio.NewScanner(strings.NewReader(responseText))
	var fullText string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			var jsonData RaycastSSEData
			if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
				log.Printf("Failed to parse SSE data: %v", err)
				continue
			}
			if jsonData.Text != "" {
				fullText += jsonData.Text
			}
		}
	}

	return fullText
}

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

	// Fetch models directly before each request
	models, err := fetchModels(config)
	if err != nil {
		log.Printf("Error fetching models: %v, using defaults", err)
		models = make(map[string]ModelCacheEntry)
	}

	// Get provider info from the fetched models
	provider, modelName := getProviderInfo(model, models)

	log.Printf("Using provider: %s, model: %s", provider, modelName)

	// Create a unique thread ID for this conversation
	threadId := uuid.New().String()

	// Prepare Raycast request
	raycastRequest := RaycastChatRequest{
		AdditionalSystemInstructions: "",
		Debug:                        false,
		Locale:                       "en-US",
		Messages:                     convertMessages(body.Messages),
		Model:                        modelName,
		Provider:                     provider,
		Source:                       "ai_chat",
		SystemInstruction:            "markdown",
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

	client := &http.Client{}
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

// handleStreamingResponse handles streaming response from Raycast
func handleStreamingResponse(c *gin.Context, response *http.Response, modelId string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)

	// Set up a flush interval for the writer
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		log.Println("Streaming unsupported")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	reader := bufio.NewReader(response.Body)
	buffer := ""

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("Error reading from response: %v", err)
			break
		}

		buffer += line

		// Process complete SSE messages in the buffer
		if strings.HasSuffix(buffer, "\n\n") {
			lines := strings.Split(buffer, "\n")
			buffer = ""

			for _, l := range lines {
				if strings.TrimSpace(l) == "" {
					continue
				}

				if strings.HasPrefix(l, "data:") {
					data := strings.TrimSpace(strings.TrimPrefix(l, "data:"))
					var jsonData RaycastSSEData
					if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
						log.Printf("Failed to parse SSE data: %v", err)
						continue
					}

					// Create OpenAI-compatible streaming chunk
					chunk := struct {
						ID      string `json:"id"`
						Object  string `json:"object"`
						Created int64  `json:"created"`
						Model   string `json:"model"`
						Choices []struct {
							Index int `json:"index"`
							Delta struct {
								Content string `json:"content"`
							} `json:"delta"`
							FinishReason string `json:"finish_reason"`
						} `json:"choices"`
					}{
						ID:      fmt.Sprintf("chatcmpl-%s", uuid.New().String()),
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Model:   modelId,
						Choices: []struct {
							Index int `json:"index"`
							Delta struct {
								Content string `json:"content"`
							} `json:"delta"`
							FinishReason string `json:"finish_reason"`
						}{
							{
								Index: 0,
								Delta: struct {
									Content string `json:"content"`
								}{
									Content: jsonData.Text,
								},
								FinishReason: jsonData.FinishReason,
							},
						},
					}

					chunkData, err := json.Marshal(chunk)
					if err != nil {
						log.Printf("Error marshaling chunk: %v", err)
						continue
					}

					// Send the chunk
					fmt.Fprintf(c.Writer, "data: %s\n\n", string(chunkData))
					flusher.Flush()
				}
			}
		}
	}

	// Send final [DONE] marker
	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
}

// handleNonStreamingResponse handles non-streaming response from Raycast
func handleNonStreamingResponse(c *gin.Context, response *http.Response, modelId string) {
	// Collect the entire response
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Details string `json:"details,omitempty"`
			}{
				Message: "Error reading response body",
				Type:    "server_error",
				Details: err.Error(),
			},
		})
		return
	}

	responseText := string(bodyBytes)
	log.Printf("Raw response: %s", responseText)

	// Parse the SSE format to extract the full text
	fullText := parseSSEResponse(responseText)

	// Convert to OpenAI format
	openaiResponse := OpenAIChatResponse{
		ID:      fmt.Sprintf("chatcmpl-%s", uuid.New().String()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   modelId,
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role        string   `json:"role"`
				Content     string   `json:"content"`
				Refusal     *string  `json:"refusal"`
				Annotations []string `json:"annotations"`
			} `json:"message"`
			Logprobs     *string `json:"logprobs"`
			FinishReason string  `json:"finish_reason"`
		}{
			{
				Index: 0,
				Message: struct {
					Role        string   `json:"role"`
					Content     string   `json:"content"`
					Refusal     *string  `json:"refusal"`
					Annotations []string `json:"annotations"`
				}{
					Role:        "assistant",
					Content:     fullText,
					Refusal:     nil,
					Annotations: []string{},
				},
				Logprobs:     nil,
				FinishReason: "length",
			},
		},
		Usage: struct {
			PromptTokens        int `json:"prompt_tokens"`
			CompletionTokens    int `json:"completion_tokens"`
			TotalTokens         int `json:"total_tokens"`
			PromptTokensDetails struct {
				CachedTokens int `json:"cached_tokens"`
				AudioTokens  int `json:"audio_tokens"`
			} `json:"prompt_tokens_details"`
			CompletionTokensDetails struct {
				ReasoningTokens          int `json:"reasoning_tokens"`
				AudioTokens              int `json:"audio_tokens"`
				AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
				RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
			} `json:"completion_tokens_details"`
		}{
			PromptTokens:     10,
			CompletionTokens: 10,
			TotalTokens:      20,
			PromptTokensDetails: struct {
				CachedTokens int `json:"cached_tokens"`
				AudioTokens  int `json:"audio_tokens"`
			}{
				CachedTokens: 0,
				AudioTokens:  0,
			},
			CompletionTokensDetails: struct {
				ReasoningTokens          int `json:"reasoning_tokens"`
				AudioTokens              int `json:"audio_tokens"`
				AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
				RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
			}{
				ReasoningTokens:          0,
				AudioTokens:              0,
				AcceptedPredictionTokens: 0,
				RejectedPredictionTokens: 0,
			},
		},
		ServiceTier:       "default",
		SystemFingerprint: "fp_b376dfbbd5",
	}

	jsonData, err := json.MarshalIndent(openaiResponse, "", "  ")
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

	// Add a newline to the end of the JSON data
	jsonData = append(jsonData, '\n')
	// Set content type and write the formatted JSON
	c.Header("Content-Type", "application/json")
	c.Writer.Write(jsonData)
}

// handleModels handles models endpoint
func handleModels(c *gin.Context, config Config) {
	// Fetch models directly
	models, err := fetchModels(config)
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

	// Convert models to OpenAI format
	openaiModels := OpenAIModelResponse{
		Object: "list",
		Data: make([]struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		}, 0, len(models)),
	}

	for _, info := range models {
		openaiModels.Data = append(openaiModels.Data, struct {
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

	// Add a newline to the end of the JSON data
	jsonData = append(jsonData, '\n')

	// Set content type and write the formatted JSON
	c.Header("Content-Type", "application/json")
	c.Writer.Write(jsonData)
}

// Main function
func main() {
	// Configure logging
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Load configuration from environment variables
	config := Config{
		RaycastBearerToken: os.Getenv("RAYCAST_BEARER_TOKEN"),
		APIKey:             os.Getenv("API_KEY"),
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

	// Route endpoints
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		handleChatCompletions(c, config)
	})

	router.GET("/v1/models", func(c *gin.Context) {
		handleModels(c, config)
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

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
