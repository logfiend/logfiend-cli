package providers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/logfiend/internal/types"
)

// SplunkProvider implements the Provider interface for Splunk
type SplunkProvider struct {
	config types.ProviderConfig
	client *http.Client
}

// SplunkIndexResponse represents Splunk's index API response
type SplunkIndexResponse struct {
	Entry []struct {
		Name    string `json:"name"`
		Content struct {
			MaxSize               string `json:"maxTotalDataSizeMB"`
			CurrentSizeMB         string `json:"currentDBSizeMB"`
			MaxTime               string `json:"maxTime"`
			MinTime               string `json:"minTime"`
			TotalEventCount       string `json:"totalEventCount"`
			IsInternal            string `json:"isInternal"`
			DataType              string `json:"datatype"`
			HomePath              string `json:"homePath"`
			ColdPath              string `json:"coldPath"`
			ThawedPath            string `json:"thawedPath"`
			EnableOnlineBucketRepair string `json:"enableOnlineBucketRepair"`
		} `json:"content"`
	} `json:"entry"`
}

// NewSplunkProvider creates a new Splunk provider
func NewSplunkProvider(config types.ProviderConfig) (types.Provider, error) {
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

	return &SplunkProvider{
		config: config,
		client: client,
	}, nil
}

func (s *SplunkProvider) Name() string {
	return "splunk"
}

func (s *SplunkProvider) FetchDataViews(ctx context.Context) ([]types.DataSource, error) {
	// Splunk uses indexes as data sources
	return s.fetchIndexes(ctx)
}

func (s *SplunkProvider) fetchIndexes(ctx context.Context) ([]types.DataSource, error) {
	// Build URL for Splunk's REST API
	baseURL := strings.TrimSuffix(s.config.Endpoint, "/")
	endpoint := fmt.Sprintf("%s/services/data/indexes", baseURL)
	
	// Add query parameters
	params := url.Values{}
	params.Add("output_mode", "json")
	params.Add("count", "0") // Get all indexes
	
	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication
	if s.config.Auth != nil {
		s.addAuth(req)
	}

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("splunk returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var splunkResp SplunkIndexResponse
	if err := json.NewDecoder(resp.Body).Decode(&splunkResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to DataSource objects
	dataSources := make([]types.DataSource, 0, len(splunkResp.Entry))
	for _, entry := range splunkResp.Entry {
		ds := s.convertToDataSource(entry)
		dataSources = append(dataSources, ds)
	}

	return dataSources, nil
}

func (s *SplunkProvider) convertToDataSource(entry struct {
	Name    string `json:"name"`
	Content struct {
		MaxSize               string `json:"maxTotalDataSizeMB"`
		CurrentSizeMB         string `json:"currentDBSizeMB"`
		MaxTime               string `json:"maxTime"`
		MinTime               string `json:"minTime"`
		TotalEventCount       string `json:"totalEventCount"`
		IsInternal            string `json:"isInternal"`
		DataType              string `json:"datatype"`
		HomePath              string `json:"homePath"`
		ColdPath              string `json:"coldPath"`
		ThawedPath            string `json:"thawedPath"`
		EnableOnlineBucketRepair string `json:"enableOnlineBucketRepair"`
	} `json:"content"`
}) types.DataSource {
	
	ds := types.DataSource{
		ID:      entry.Name,
		Name:    entry.Name,
		Title:   entry.Name,
		Type:    "splunk-index",
		Pattern: fmt.Sprintf("index=%s", entry.Name),
	}

	// Determine status based on internal flag
	if entry.Content.IsInternal == "1" {
		ds.Status = "internal"
		ds.Tags = append(ds.Tags, "internal")
	} else {
		ds.Status = "active"
		ds.Tags = append(ds.Tags, "external")
	}

	// Add metadata
	ds.Metadata = map[string]interface{}{
		"maxSizeMB":         entry.Content.MaxSize,
		"currentSizeMB":     entry.Content.CurrentSizeMB,
		"totalEventCount":   entry.Content.TotalEventCount,
		"dataType":          entry.Content.DataType,
		"homePath":          entry.Content.HomePath,
		"coldPath":          entry.Content.ColdPath,
		"thawedPath":        entry.Content.ThawedPath,
	}

	// Parse time ranges if available
	if entry.Content.MinTime != "" && entry.Content.MinTime != "0" {
		if minTime, err := time.Parse("2006-01-02T15:04:05.000-07:00", entry.Content.MinTime); err == nil {
			ds.CreatedAt = &minTime
		}
	}

	return ds
}

func (s *SplunkProvider) addAuth(req *http.Request) {
	auth := s.config.Auth
	switch auth.Type {
	case "basic":
		req.SetBasicAuth(auth.Username, auth.Password)
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+auth.Token)
	case "api_key":
		req.Header.Set("Authorization", "Splunk "+auth.APIKey)
	}
}

func (s *SplunkProvider) ValidateConnection(ctx context.Context) error {
	baseURL := strings.TrimSuffix(s.config.Endpoint, "/")
	url := fmt.Sprintf("%s/services/server/info?output_mode=json", baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	if s.config.Auth != nil {
		s.addAuth(req)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Splunk: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("splunk health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (s *SplunkProvider) GetCapabilities() types.ProviderCapabilities {
	return types.ProviderCapabilities{
		SupportsRealTimeQueries: true,
		SupportsHistoricalData:  true,
		SupportedDataTypes:      []string{"splunk-index", "summary-index"},
		RequiresAuthentication:  s.config.Auth != nil,
	}
}
