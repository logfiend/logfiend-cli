package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/logfiend/internal/types"
	"gopkg.in/yaml.v3"
)

// Config represents the main application configuration
type Config struct {
	Provider types.ProviderConfig `yaml:"provider"`
	Output   OutputConfig         `yaml:"output,omitempty"`
	Logging  LoggingConfig        `yaml:"logging,omitempty"`
}

// OutputConfig configures output settings
type OutputConfig struct {
	Format    string `yaml:"format,omitempty"`     // json, yaml, csv
	Pretty    bool   `yaml:"pretty,omitempty"`     // pretty print JSON
	Timestamp bool   `yaml:"timestamp,omitempty"`  // include timestamp in filename
}

// LoggingConfig configures logging settings
type LoggingConfig struct {
	Level  string `yaml:"level,omitempty"`  // debug, info, warn, error
	Format string `yaml:"format,omitempty"` // text, json
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	// Validate and sanitize path
	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) {
		return nil, fmt.Errorf("absolute paths not allowed for security")
	}

	// Set defaults
	cfg := &Config{
		Provider: types.ProviderConfig{
			Timeout: 30 * time.Second,
			Retries: 3,
		},
		Output: OutputConfig{
			Format:    "json",
			Pretty:    true,
			Timestamp: false,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}

	// Read file
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file '%s': %w", cleanPath, err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("error parsing YAML config: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Provider.Type == "" {
		return fmt.Errorf("provider type is required")
	}
	if c.Provider.Endpoint == "" {
		return fmt.Errorf("provider endpoint is required")
	}

	// Validate auth config if present
	if c.Provider.Auth != nil {
		if err := c.validateAuth(); err != nil {
			return fmt.Errorf("invalid auth config: %w", err)
		}
	}

	return nil
}

func (c *Config) validateAuth() error {
	auth := c.Provider.Auth
	switch auth.Type {
	case "basic":
		if auth.Username == "" || auth.Password == "" {
			return fmt.Errorf("basic auth requires username and password")
		}
	case "bearer":
		if auth.Token == "" {
			return fmt.Errorf("bearer auth requires token")
		}
	case "api_key":
		if auth.APIKey == "" {
			return fmt.Errorf("api_key auth requires api_key")
		}
	default:
		return fmt.Errorf("unsupported auth type: %s", auth.Type)
	}
	return nil
}

// Sanitize cleans and validates configuration values
func (c *Config) Sanitize() error {
	// Sanitize endpoint URL
	if err := c.sanitizeEndpoint(); err != nil {
		return fmt.Errorf("invalid endpoint: %w", err)
	}

	// Sanitize provider type
	c.Provider.Type = strings.ToLower(strings.TrimSpace(c.Provider.Type))

	// Sanitize auth fields if present
	if c.Provider.Auth != nil {
		c.Provider.Auth.Username = strings.TrimSpace(c.Provider.Auth.Username)
		c.Provider.Auth.Type = strings.ToLower(strings.TrimSpace(c.Provider.Auth.Type))

		// Never log or expose password/token/key values
		// Validate they exist but don't process their content
		if c.Provider.Auth.Type == "basic" && (c.Provider.Auth.Username == "" || c.Provider.Auth.Password == "") {
			return fmt.Errorf("basic auth requires non-empty username and password")
		}
		if c.Provider.Auth.Type == "bearer" && c.Provider.Auth.Token == "" {
			return fmt.Errorf("bearer auth requires non-empty token")
		}
		if c.Provider.Auth.Type == "api_key" && c.Provider.Auth.APIKey == "" {
			return fmt.Errorf("api_key auth requires non-empty api_key")
		}
	}

	return nil
}

func (c *Config) sanitizeEndpoint() error {
	endpoint := strings.TrimSpace(c.Provider.Endpoint)
	if endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty")
	}

	// Basic URL validation
	urlPattern := regexp.MustCompile(`^https?://[a-zA-Z0-9\-\.]+(:[0-9]+)?(/.*)?$`)
	if !urlPattern.MatchString(endpoint) {
		return fmt.Errorf("endpoint must be a valid HTTP/HTTPS URL")
	}

	// Ensure HTTPS for production (allow HTTP only for localhost/127.0.0.1)
	if strings.HasPrefix(endpoint, "http://") {
		if !strings.Contains(endpoint, "localhost") && !strings.Contains(endpoint, "127.0.0.1") {
			return fmt.Errorf("HTTP endpoints only allowed for localhost/127.0.0.1, use HTTPS for remote endpoints")
		}
	}

	c.Provider.Endpoint = endpoint
	return nil
}
