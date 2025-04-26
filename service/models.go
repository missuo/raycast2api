/*
 * @Author: Vincent Yang
 * @Date: 2025-04-08 22:43:35
 * @LastEditors: Vincent Yang
 * @LastEditTime: 2025-04-26 17:21:02
 * @FilePath: /raycast2api/service/models.go
 * @Telegram: https://t.me/missuo
 * @GitHub: https://github.com/missuo
 *
 * Copyright © 2025 by Vincent, All Rights Reserved.
 */

package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// NewModelCache creates a new model cache
func NewModelCache() *ModelCache {
	return &ModelCache{
		models:    make(map[string]ModelCacheEntry),
		expiresAt: time.Now(),
		mutex:     sync.RWMutex{},
	}
}

// GetModels gets models from cache or fetches them from Raycast API
func (mc *ModelCache) GetModels(config Config) (map[string]ModelCacheEntry, error) {
	mc.mutex.RLock()
	if time.Now().Before(mc.expiresAt) && len(mc.models) > 0 {
		defer mc.mutex.RUnlock()
		log.Println("Using cached models")
		return mc.models, nil
	}
	mc.mutex.RUnlock()

	// Cache has expired or is empty, fetch new data
	models, err := fetchModelsFromAPI(config)
	if err != nil {
		log.Printf("Error fetching models: %v, using defaults or cached data", err)
		mc.mutex.RLock()
		defer mc.mutex.RUnlock()

		// If we have cached models, return them even if expired
		if len(mc.models) > 0 {
			log.Println("Using expired cached models as fallback")
			return mc.models, nil
		}

		// If no cached models, create a default entry
		defaultModels := map[string]ModelCacheEntry{
			DefaultModel: {
				Provider: DefaultProvider,
				Model:    DefaultModel,
			},
		}
		return defaultModels, err
	}

	// Update the cache with new data
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.models = models
	mc.expiresAt = time.Now().Add(ModelCacheTTL)
	log.Printf("Model cache updated with %d models, expires at %v", len(models), mc.expiresAt)

	return models, nil
}

// ForceCacheRefresh forces a refresh of the model cache
func (mc *ModelCache) ForceCacheRefresh(config Config) {
	mc.mutex.Lock()
	mc.expiresAt = time.Now() // Expire the cache
	mc.mutex.Unlock()

	// Trigger a refresh
	_, _ = mc.GetModels(config)
}

// fetchModelsFromAPI fetches model information from Raycast API
func fetchModelsFromAPI(config Config) (map[string]ModelCacheEntry, error) {
	log.Println("Fetching models from Raycast API...")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
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
		return nil, fmt.Errorf("raycast api error: %d %s", resp.StatusCode, string(bodyBytes))
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

// getProviderInfo gets provider info for a model
func getProviderInfo(modelID string, models map[string]ModelCacheEntry) (string, string) {
	if model, ok := models[modelID]; ok {
		return model.Provider, model.Model
	}
	// Fallback to defaults
	return DefaultProvider, DefaultModel
}
