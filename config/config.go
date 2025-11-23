package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	SPA      SPAConfig
	App      AppConfig
	LLM      LLMConfig
	CORS     CORSConfig
}

type ServerConfig struct {
	Port string
	Env  string
}

type DatabaseConfig struct {
	URL string
}

type SPAConfig struct {
	Mode string
	Dir  string
}

type AppConfig struct {
	DefaultTimezone string
}

type LLMConfig struct {
	Provider           string
	OllamaBaseURL      string
	EmbeddingModel     string
	ChatModel          string
	VectorDimensions   int
	UpdateInterval     time.Duration
	EnableVectorSearch bool
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// Load loads configuration with the following priority:
// 1. Environment variables (highest priority)
// 2. Config file (user config dir or --config flag)
// 3. Defaults (lowest priority)
func Load() *Config {
	var cfg *Config

	// Try to load from config file
	configPath, err := GetConfigPath()
	if err == nil && fileExists(configPath) {
		cfg, err = LoadFromFile(configPath)
		if err != nil {
			// Config file exists but is invalid - use defaults
			cfg = getDefaultConfig()
		}
	} else {
		// No config file - use defaults
		cfg = getDefaultConfig()
	}

	// Override with environment variables (highest priority)
	if envPort := os.Getenv("PORT"); envPort != "" {
		cfg.Server.Port = envPort
	}
	if envEnv := os.Getenv("APP_ENV"); envEnv != "" {
		cfg.Server.Env = envEnv
	}
	if envDB := os.Getenv("DATABASE_URL"); envDB != "" {
		cfg.Database.URL = envDB
	}
	if envOllama := os.Getenv("OLLAMA_BASE_URL"); envOllama != "" {
		cfg.LLM.OllamaBaseURL = envOllama
	}
	if envEmbedding := os.Getenv("EMBEDDING_MODEL"); envEmbedding != "" {
		cfg.LLM.EmbeddingModel = envEmbedding
	}
	if envChat := os.Getenv("CHAT_MODEL"); envChat != "" {
		cfg.LLM.ChatModel = envChat
	}
	if envVecSearch := os.Getenv("ENABLE_VECTOR_SEARCH"); envVecSearch != "" {
		if val, err := strconv.ParseBool(envVecSearch); err == nil {
			cfg.LLM.EnableVectorSearch = val
		}
	}
	if envCORS := os.Getenv("CORS_ORIGINS"); envCORS != "" {
		cfg.CORS.AllowedOrigins = parseCORSOrigins(envCORS)
	}

	return cfg
}

func getDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: "8080",
			Env:  "development",
		},
		Database: DatabaseConfig{
			URL: "",
		},
		SPA: SPAConfig{
			Mode: "fs",
			Dir:  "web/dist",
		},
		App: AppConfig{
			DefaultTimezone: "America/New_York",
		},
		LLM: LLMConfig{
			Provider:           "ollama",
			OllamaBaseURL:      "http://localhost:11434",
			EmbeddingModel:     "nomic-embed-text",
			ChatModel:          "llama3.2",
			VectorDimensions:   768,
			UpdateInterval:     60 * time.Second,
			EnableVectorSearch: true,
		},
		CORS: CORSConfig{
			AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:8080"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		},
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return fallback
}

func parseCORSOrigins(origins string) []string {
	if origins == "" {
		return []string{}
	}
	parts := strings.Split(origins, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// GetUserConfigDir returns the OS-specific user configuration directory
func GetUserConfigDir() (string, error) {
	var configDir string

	// Get OS-specific config directory
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		// Linux with XDG
		configDir = dir
	} else if home, err := os.UserHomeDir(); err == nil {
		// macOS, Linux, Windows
		switch {
		case fileExists(filepath.Join(home, "Library")):
			// macOS
			configDir = filepath.Join(home, "Library", "Application Support")
		case os.Getenv("APPDATA") != "":
			// Windows
			configDir = os.Getenv("APPDATA")
		default:
			// Linux/Unix
			configDir = filepath.Join(home, ".config")
		}
	} else {
		return "", fmt.Errorf("unable to determine home directory: %w", err)
	}

	// Add app-specific subdirectory
	appConfigDir := filepath.Join(configDir, "journal")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return appConfigDir, nil
}

// GetConfigPath returns the full path to the config file
func GetConfigPath() (string, error) {
	// Check for --config flag or CONFIG_FILE env var first
	if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
		return configFile, nil
	}

	// Check for .env in current directory (backward compatibility)
	if fileExists(".env") {
		return ".env", nil
	}

	// Use user config directory
	configDir, err := GetUserConfigDir()
	if err != nil {
		return "", err
	}

	// Try YAML first, then fall back to .env format
	yamlPath := filepath.Join(configDir, "config.yaml")
	envPath := filepath.Join(configDir, "config.env")

	if fileExists(yamlPath) {
		return yamlPath, nil
	}
	if fileExists(envPath) {
		return envPath, nil
	}

	// Return yaml path as default for new installations
	return yamlPath, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// LoadFromFile loads configuration from a YAML or ENV file
func LoadFromFile(path string) (*Config, error) {
	// Check if file exists
	if !fileExists(path) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Determine format based on extension
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		return loadYAML(data)
	}

	// Fall back to environment variable format
	return loadEnvFormat(data)
}

func loadYAML(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Apply defaults
	applyDefaults(&cfg)

	return &cfg, nil
}

func loadEnvFormat(data []byte) (*Config, error) {
	// Parse .env format line by line
	lines := strings.Split(string(data), "\n")
	envMap := make(map[string]string)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			envMap[key] = value
		}
	}

	// Build config from map
	cfg := &Config{
		Server: ServerConfig{
			Port: getFromMap(envMap, "PORT", "8080"),
			Env:  getFromMap(envMap, "APP_ENV", "development"),
		},
		Database: DatabaseConfig{
			URL: getFromMap(envMap, "DATABASE_URL", ""),
		},
		SPA: SPAConfig{
			Mode: getFromMap(envMap, "SPA_MODE", "fs"),
			Dir:  getFromMap(envMap, "SPA_DIR", "web/dist"),
		},
		App: AppConfig{
			DefaultTimezone: getFromMap(envMap, "DEFAULT_TIMEZONE", "America/New_York"),
		},
		LLM: LLMConfig{
			Provider:           getFromMap(envMap, "LLM_PROVIDER", "ollama"),
			OllamaBaseURL:      getFromMap(envMap, "OLLAMA_BASE_URL", "http://localhost:11434"),
			EmbeddingModel:     getFromMap(envMap, "EMBEDDING_MODEL", "nomic-embed-text"),
			ChatModel:          getFromMap(envMap, "CHAT_MODEL", "llama3.2"),
			VectorDimensions:   getIntFromMap(envMap, "VECTOR_DIMENSIONS", 768),
			UpdateInterval:     time.Duration(getIntFromMap(envMap, "VECTOR_UPDATE_INTERVAL", 60)) * time.Second,
			EnableVectorSearch: getBoolFromMap(envMap, "ENABLE_VECTOR_SEARCH", true),
		},
		CORS: CORSConfig{
			AllowedOrigins:   parseCORSOrigins(getFromMap(envMap, "CORS_ORIGINS", "http://localhost:5173,http://localhost:8080")),
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		},
	}

	return cfg, nil
}

func getFromMap(m map[string]string, key, fallback string) string {
	if value, ok := m[key]; ok && value != "" {
		return value
	}
	return fallback
}

func getIntFromMap(m map[string]string, key string, fallback int) int {
	if value, ok := m[key]; ok && value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func getBoolFromMap(m map[string]string, key string, fallback bool) bool {
	if value, ok := m[key]; ok && value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return fallback
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}
	if cfg.Server.Env == "" {
		cfg.Server.Env = "development"
	}
	if cfg.SPA.Mode == "" {
		cfg.SPA.Mode = "fs"
	}
	if cfg.SPA.Dir == "" {
		cfg.SPA.Dir = "web/dist"
	}
	if cfg.App.DefaultTimezone == "" {
		cfg.App.DefaultTimezone = "America/New_York"
	}
	if cfg.LLM.Provider == "" {
		cfg.LLM.Provider = "ollama"
	}
	if cfg.LLM.OllamaBaseURL == "" {
		cfg.LLM.OllamaBaseURL = "http://localhost:11434"
	}
	if cfg.LLM.EmbeddingModel == "" {
		cfg.LLM.EmbeddingModel = "nomic-embed-text"
	}
	if cfg.LLM.ChatModel == "" {
		cfg.LLM.ChatModel = "llama3.2"
	}
	if cfg.LLM.VectorDimensions == 0 {
		cfg.LLM.VectorDimensions = 768
	}
	if cfg.LLM.UpdateInterval == 0 {
		cfg.LLM.UpdateInterval = 60 * time.Second
	}
	if len(cfg.CORS.AllowedOrigins) == 0 {
		cfg.CORS.AllowedOrigins = []string{"http://localhost:5173", "http://localhost:8080"}
	}
	if cfg.CORS.MaxAge == 0 {
		cfg.CORS.MaxAge = 12 * time.Hour
	}
	cfg.CORS.AllowCredentials = true
}

// SaveConfigFile writes the configuration to a file
func SaveConfigFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0600)
}
