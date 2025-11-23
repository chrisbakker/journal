package config

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

// ValidationResult contains validation results and errors
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// Validate validates the configuration
func (c *Config) Validate() *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}

	// Validate Database URL
	if c.Database.URL == "" {
		result.addError("DATABASE_URL", "Database URL is required")
	} else {
		if err := validateDatabaseURL(c.Database.URL); err != nil {
			result.addError("DATABASE_URL", err.Error())
		}
	}

	// Validate Ollama URL if vector search is enabled
	if c.LLM.EnableVectorSearch {
		if c.LLM.OllamaBaseURL == "" {
			result.addError("OLLAMA_BASE_URL", "Ollama base URL is required when vector search is enabled")
		} else {
			if _, err := url.Parse(c.LLM.OllamaBaseURL); err != nil {
				result.addError("OLLAMA_BASE_URL", "Invalid Ollama URL format")
			}
		}

		if c.LLM.EmbeddingModel == "" {
			result.addError("EMBEDDING_MODEL", "Embedding model is required when vector search is enabled")
		}

		if c.LLM.ChatModel == "" {
			result.addError("CHAT_MODEL", "Chat model is required when vector search is enabled")
		}
	}

	return result
}

func (r *ValidationResult) addError(field, message string) {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// validateDatabaseURL validates the database connection string format
func validateDatabaseURL(dbURL string) error {
	// Parse URL
	u, err := url.Parse(dbURL)
	if err != nil {
		return fmt.Errorf("invalid database URL format: %w", err)
	}

	if u.Scheme != "postgresql" && u.Scheme != "postgres" {
		return fmt.Errorf("database URL must use postgresql:// or postgres:// scheme")
	}

	if u.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if u.Path == "" || u.Path == "/" {
		return fmt.Errorf("database name is required")
	}

	return nil
}

// CheckEnvFile checks if a config file exists
func CheckEnvFile() (bool, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return false, err
	}

	return fileExists(configPath), nil
}

// FormatErrorsForDisplay formats validation errors for display
func (r *ValidationResult) FormatErrorsForDisplay() string {
	if r.Valid {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Configuration errors found:\n\n")
	for _, err := range r.Errors {
		sb.WriteString(fmt.Sprintf("• %s: %s\n", err.Field, err.Message))
	}
	return sb.String()
}
