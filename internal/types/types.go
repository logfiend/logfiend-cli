package types

import (
	"context"
	"time"
)

// DataSource represents a data source entity in any SIEM system
type DataSource struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Title       string                 `json:"title"`
	Type        string                 `json:"type"`
	Pattern     string                 `json:"pattern,omitempty"`
	Description string                 `json:"description,omitempty"`
	CreatedAt   *time.Time             `json:"created_at,omitempty"`
	UpdatedAt   *time.Time             `json:"updated_at,omitempty"`
	Status      string                 `json:"status,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// InventoryMetadata contains metadata about the inventory collection
type InventoryMetadata struct {
	Timestamp   time.Time `json:"timestamp"`
	Provider    string    `json:"provider"`
	Version     string    `json:"version"`
	SourceCount int       `json:"source_count"`
	GeneratedBy string    `json:"generated_by"`
}

// DataSourceInventory holds the complete inventory with metadata
type DataSourceInventory struct {
	Metadata    InventoryMetadata `json:"metadata"`
	DataSources []DataSource      `json:"data_sources"`
}

// Provider defines the interface that all SIEM providers must implement
// This follows the OpenTelemetry pattern of vendor-agnostic interfaces
type Provider interface {
	// Name returns the provider identifier
	Name() string
	
	// FetchDataSources retrieves all data sources from the SIEM
	FetchDataViews(ctx context.Context) ([]DataSource, error)
	
	// ValidateConnection tests the connection to the SIEM
	ValidateConnection(ctx context.Context) error
	
	// GetCapabilities returns what features this provider supports
	GetCapabilities() ProviderCapabilities
}

// ProviderCapabilities describes what features a provider supports
type ProviderCapabilities struct {
	SupportsRealTimeQueries bool     `json:"supports_real_time_queries"`
	SupportsHistoricalData  bool     `json:"supports_historical_data"`
	SupportedDataTypes      []string `json:"supported_data_types"`
	RequiresAuthentication  bool     `json:"requires_authentication"`
}

// ProviderConfig holds configuration for any provider
type ProviderConfig struct {
	Type      string            `yaml:"type" json:"type"`
	Endpoint  string            `yaml:"endpoint" json:"endpoint"`
	Options   map[string]string `yaml:"options,omitempty" json:"options,omitempty"`
	Auth      *AuthConfig       `yaml:"auth,omitempty" json:"auth,omitempty"`
	TLS       *TLSConfig        `yaml:"tls,omitempty" json:"tls,omitempty"`
	Timeout   time.Duration     `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retries   int               `yaml:"retries,omitempty" json:"retries,omitempty"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Type     string `yaml:"type" json:"type"` // basic, bearer, api_key
	Username string `yaml:"username,omitempty" json:"username,omitempty"`
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
	Token    string `yaml:"token,omitempty" json:"token,omitempty"`
	APIKey   string `yaml:"api_key,omitempty" json:"api_key,omitempty"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled            bool   `yaml:"enabled" json:"enabled"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify,omitempty" json:"insecure_skip_verify,omitempty"`
	CertFile           string `yaml:"cert_file,omitempty" json:"cert_file,omitempty"`
	KeyFile            string `yaml:"key_file,omitempty" json:"key_file,omitempty"`
	CAFile             string `yaml:"ca_file,omitempty" json:"ca_file,omitempty"`
}
