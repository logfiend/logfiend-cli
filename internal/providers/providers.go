package providers

import (
	"fmt"
	"strings"

	"github.com/logfiend/internal/types"
)

// ProviderFactory is a function that creates a new provider instance
type ProviderFactory func(config types.ProviderConfig) (types.Provider, error)

// Registry holds all registered provider factories
var registry = make(map[string]ProviderFactory)

// Register adds a new provider factory to the registry
func Register(name string, factory ProviderFactory) {
	registry[strings.ToLower(name)] = factory
}

// NewProvider creates a new provider instance based on the configuration
func NewProvider(config types.ProviderConfig) (types.Provider, error) {
	factory, exists := registry[strings.ToLower(config.Type)]
	if !exists {
		return nil, fmt.Errorf("unsupported provider type: %s (available: %v)", 
			config.Type, getAvailableProviders())
	}

	return factory(config)
}

// GetAvailableProviders returns a list of registered provider names
func GetAvailableProviders() []string {
	return getAvailableProviders()
}

func getAvailableProviders() []string {
	providers := make([]string, 0, len(registry))
	for name := range registry {
		providers = append(providers, name)
	}
	return providers
}

// init registers all built-in providers
func init() {
	Register("elasticsearch", NewElasticsearchProvider)
	Register("splunk", NewSplunkProvider)
	Register("sentinel", NewSentinelProvider)
	Register("qradar", NewQRadarProvider)
}
