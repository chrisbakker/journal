package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/chrisbakker/journal/config"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// ConfigSetupRequest represents the configuration setup form data
type ConfigSetupRequest struct {
	DatabaseHost     string `json:"database_host" binding:"required"`
	DatabasePort     string `json:"database_port" binding:"required"`
	DatabaseName     string `json:"database_name" binding:"required"`
	DatabaseUser     string `json:"database_user" binding:"required"`
	DatabasePassword string `json:"database_password" binding:"required"`
	DatabaseSSLMode  string `json:"database_ssl_mode"`
	OllamaBaseURL    string `json:"ollama_base_url" binding:"required"`
	EmbeddingModel   string `json:"embedding_model" binding:"required"`
	ChatModel        string `json:"chat_model" binding:"required"`
}

// Reloader interface for dependency injection
type Reloader interface {
	Reload() error
}

// SaveConfig handles saving the configuration and triggering internal reload
func (h *Handler) SaveConfig(c *gin.Context, reloader Reloader) {
	var req ConfigSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default SSL mode to disable if not provided
	if req.DatabaseSSLMode == "" {
		req.DatabaseSSLMode = "disable"
	}

	// Build DATABASE_URL
	databaseURL := fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		req.DatabaseUser,
		req.DatabasePassword,
		req.DatabaseHost,
		req.DatabasePort,
		req.DatabaseName,
		req.DatabaseSSLMode,
	)

	// Build config structure
	cfg := config.Config{
		Server: config.ServerConfig{
			Port: "8080",
			Env:  "development",
		},
		Database: config.DatabaseConfig{
			URL: databaseURL,
		},
		SPA: config.SPAConfig{
			Mode: "fs",
			Dir:  "web/dist",
		},
		App: config.AppConfig{
			DefaultTimezone: "America/New_York",
		},
		LLM: config.LLMConfig{
			Provider:           "ollama",
			OllamaBaseURL:      req.OllamaBaseURL,
			EmbeddingModel:     req.EmbeddingModel,
			ChatModel:          req.ChatModel,
			VectorDimensions:   768,
			UpdateInterval:     60 * time.Second,
			EnableVectorSearch: true,
		},
		CORS: config.CORSConfig{
			AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:8080"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		},
	}

	// Get config file path
	configPath, err := config.GetConfigPath()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to determine config file location"})
		return
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(&cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate configuration"})
		return
	}

	// Write to config file
	if err := config.SaveConfigFile(configPath, yamlData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save configuration: %v", err)})
		return
	}

	// Reload configuration and reconnect to resources
	// This happens in the background so the response can be sent immediately
	go func() {
		time.Sleep(500 * time.Millisecond) // Give time for response to be sent
		if err := reloader.Reload(); err != nil {
			log.Printf("❌ Failed to reload configuration: %v\n", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":    fmt.Sprintf("Configuration saved to %s and reloaded successfully!", configPath),
		"reload":     true,
		"configPath": configPath,
	})
}
