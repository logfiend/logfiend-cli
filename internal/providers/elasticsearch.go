package providers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/logfiend/internal/types"
)

// ElasticsearchProvider implements the Provider interface for Elasticsearch/Kibana
type ElasticsearchProvider struct {
	config types.ProviderConfig
	client *http.Client
}

// ElasticsearchResponse represents the structure of Elasticsearch search responses
type ElasticsearchResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []struct {
			ID     string `json:"_id"`
			Source struct {
				Type         string                 `json:"type"`
				IndexPattern map[string]interface{} `json:"index-pattern,omitempty"`
				DataView     map[string]interface{} `json:"data-view,omitempty"`
				UpdatedAt    string                 `json:"updated_at,omitempty"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

// NewElasticsearchProvider creates a new Elasticsearch provider
func NewElasticsearchProvider(config types.ProviderConfig) (types.Provider, error) {
	// Create HTTP client with timeout and TLS config
	client := &http.Client{
		Timeout: config.Timeout,
	}

	// Configure TLS if specified
	if config.TLS != nil && config.TLS.Enabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.TLS.InsecureSkipVerify,
		}
		client.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	return &ElasticsearchProvider{
		config: config,
		client: client,
	}, nil
}

func (e *ElasticsearchProvider) Name() string {
	return "elasticsearch"
}

func (e *ElasticsearchProvider) FetchDataViews(ctx context.Context) ([]types.DataSource, error) {
	// Try to fetch both index patterns and data views
	dataSources := []types.DataSource{}

	// Fetch index patterns (older Kibana versions)
	indexPatterns, err := e.fetchIndexPatterns(ctx)
	if err == nil {
		dataSources = append(dataSources, indexPatterns...)
	}

	// Fetch data views (newer Kibana versions)
	dataViews, err := e.fetchDataViews(ctx)
	if err == nil {
		dataSources = append(dataSources, dataViews...)
	}

	if len(dataSources) == 0 && err != nil {
		return nil, fmt.Errorf("failed to fetch any data sources: %w", err)
	}

	return dataSources, nil
}

func (e *ElasticsearchProvider) fetchIndexPatterns(ctx context.Context) ([]types.DataSource, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"type": "index-pattern",
			},
		},
		"size": 1000,
	}

	return e.executeSearch(ctx, ".kibana/_search", query, "index-pattern")
}

func (e *ElasticsearchProvider) fetchDataViews(ctx context.Context) ([]types.DataSource, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"type": "data-view",
			},
		},
		"size": 1000,
	}

	return e.executeSearch(ctx, ".kibana/_search", query, "data-view")
}

func (e *ElasticsearchProvider) executeSearch(ctx context.Context, endpoint string, query map[string]interface{}, sourceType string) ([]types.DataSource, error) {
	// Prepare request body
	bodyBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Create request
	url := strings.TrimSuffix(e.config.Endpoint, "/") + "/" + endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication if configured
	if e.config.Auth != nil {
		e.addAuth(req)
	}

	// Execute request
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("elasticsearch returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var esResp ElasticsearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&esResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to DataSource objects
	dataSources := make([]types.DataSource, 0, len(esResp.Hits.Hits))
	for _, hit := range esResp.Hits.Hits {
		ds := e.convertToDataSource(hit, sourceType)
		dataSources = append(dataSources, ds)
	}

	return dataSources, nil
}

func (e *ElasticsearchProvider) convertToDataSource(hit struct {
	ID     string `json:"_id"`
	Source struct {
		Type         string                 `json:"type"`
		IndexPattern map[string]interface{} `json:"index-pattern,omitempty"`
		DataView     map[string]interface{} `json:"data-view,omitempty"`
		UpdatedAt    string                 `json:"updated_at,omitempty"`
	} `json:"_source"`
}, sourceType string) types.DataSource {
	
	var attributes map[string]interface{}
	if sourceType == "index-pattern" && hit.Source.IndexPattern != nil {
		attributes = hit.Source.IndexPattern
	} else if sourceType == "data-view" && hit.Source.DataView != nil {
		attributes = hit.Source.DataView
	}

	ds := types.DataSource{
		ID:   hit.ID,
		Type: sourceType,
	}

	// Extract common fields
	if title, ok := attributes["title"].(string); ok {
		ds.Title = title
		ds.Name = title
		ds.Pattern = title
	}

	if timeField, ok := attributes["timeFieldName"].(string); ok {
		if ds.Metadata == nil {
			ds.Metadata = make(map[string]interface{})
		}
		ds.Metadata["timeField"] = timeField
	}

	// Parse timestamp
	if hit.Source.UpdatedAt != "" {
		if updatedAt, err := time.Parse(time.RFC3339, hit.Source.UpdatedAt); err == nil {
			ds.UpdatedAt = &updatedAt
		}
	}

	return ds
}

func (e *ElasticsearchProvider) addAuth(req *http.Request) {
	auth := e.config.Auth
	switch auth.Type {
	case "basic":
		req.SetBasicAuth(auth.Username, auth.Password)
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+auth.Token)
	case "api_key":
		req.Header.Set("Authorization", "ApiKey "+auth.APIKey)
	}
}

func (e *ElasticsearchProvider) ValidateConnection(ctx context.Context) error {
	url := strings.TrimSuffix(e.config.Endpoint, "/") + "/_aliases"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	if e.config.Auth != nil {
		e.addAuth(req)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Elasticsearch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("elasticsearch health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (e *ElasticsearchProvider) GetCapabilities() types.ProviderCapabilities {
	return types.ProviderCapabilities{
		SupportsRealTimeQueries: true,
		SupportsHistoricalData:  true,
		SupportedDataTypes:      []string{"index-pattern", "data-view"},
		RequiresAuthentication:  e.config.Auth != nil,
	}
}
