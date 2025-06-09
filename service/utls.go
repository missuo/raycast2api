/*
 * @Author: Vincent Yang
 * @Date: 2025-04-08 22:44:55
 * @LastEditors: Vincent Yang
 * @LastEditTime: 2025-04-09 15:39:59
 * @FilePath: /raycast2api/service/utls.go
 * @Telegram: https://t.me/missuo
 * @GitHub: https://github.com/missuo
 *
 * Copyright © 2025 by Vincent, All Rights Reserved.
 */

package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ConvertMessagesResult represents the result of converting OpenAI messages
type ConvertMessagesResult struct {
	RaycastMessages   []RaycastMessage
	SystemInstruction string
}

// convertMessages converts OpenAI messages format to Raycast format and extracts system instruction
func convertMessages(openaiMessages []OpenAIMessage) ConvertMessagesResult {
	systemInstruction := "markdown" // Default
	var raycastMessages []RaycastMessage

	for i, msg := range openaiMessages {
		if msg.Role == "system" && i == 0 {
			// Extract the first system message as system instruction
			switch content := msg.Content.(type) {
			case string:
				systemInstruction = content
			case []interface{}:
				// Handle array content (extract text parts)
				var contentText string
				for _, part := range content {
					if partMap, ok := part.(map[string]interface{}); ok {
						if partMap["type"] == "text" {
							if textValue, ok := partMap["text"].(string); ok {
								contentText += textValue
							}
						}
					}
				}
				if contentText != "" {
					systemInstruction = contentText
				}
			}
		} else if msg.Role == "user" || msg.Role == "assistant" {
			// Only include user and assistant messages in the messages array
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

			raycastMessages = append(raycastMessages, RaycastMessage{
				Author: author,
				Content: struct {
					Text string `json:"text"`
				}{
					Text: contentText,
				},
			})
		}
		// Ignore other roles or subsequent system messages
	}

	return ConvertMessagesResult{
		RaycastMessages:   raycastMessages,
		SystemInstruction: systemInstruction,
	}
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
