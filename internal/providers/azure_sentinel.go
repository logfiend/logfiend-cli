package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/logfiend/internal/types"
)

// SentinelProvider implements the Provider interface for Azure Sentinel
type SentinelProvider struct {
	config types.ProviderConfig
	client *http.Client
}

// SentinelTablesResponse represents Azure Log Analytics tables response
type SentinelTablesResponse struct {
	Value []struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Type       string `json:"type"`
		Properties struct {
			RetentionInDays int    `json:"retentionInDays"`
			TotalRetention  int    `json:"totalRetentionInDays"`
			ArchiveRetention int   `json:"archiveRetentionInDays"`
			Plan            string `json:"plan"`
			Schema          struct {
				Name        string `json:"name"`
				DisplayName string `json:"displayName"`
				Description string `json:"description"`
				Columns     []struct {
					Name        string `json:"name"`
					Type        string `json:"type"`
					Description string `json:"description"`
				} `json:"columns"`
			} `json:"schema"`
		} `json:"properties"`
	} `json:"value"`
}

// NewSentinelProvider creates a new Azure Sentinel provider
func NewSentinelProvider(config types.ProviderConfig) (types.Provider, error) {
	client := &http.Client{
		Timeout: config.Timeout,
	}

	return &SentinelProvider{
		config: config,
		client: client,
	}, nil
}

func (s *SentinelProvider) Name() string {
	return "azure-sentinel"
}

func (s *SentinelProvider) FetchDataViews(ctx context.Context) ([]types.DataSource, error) {
	// Extract workspace info from endpoint
	// Expected format: https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.OperationalInsights/workspaces/{workspaceName}
	workspaceInfo, err := s.parseWorkspaceFromEndpoint()
	if err != nil {
		return nil, fmt.Errorf("failed to parse workspace info: %w", err)
	}

	return s.fetchTables(ctx, workspaceInfo)
}

func (s *SentinelProvider) parseWorkspaceFromEndpoint() (map[string]string, error) {
	// Parse the Azure Resource Manager URL
	u, err := url.Parse(s.config.Endpoint)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 8 {
		return nil, fmt.Errorf("invalid Azure workspace URL format")
	}

	return map[string]string{
		"subscriptionId":    parts[1],
		"resourceGroupName": parts[3],
		"workspaceName":     parts[7],
	}, nil
}

func (s *SentinelProvider) fetchTables(ctx context.Context, workspaceInfo map[string]string) ([]types.DataSource, error) {
	// Build URL for Azure Log Analytics tables API
	apiURL := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.OperationalInsights/workspaces/%s/tables",
		workspaceInfo["subscriptionId"],
		workspaceInfo["resourceGroupName"],
		workspaceInfo["workspaceName"])

	// Add query parameters
	params := url.Values{}
	params.Add("api-version", "2022-10-01")
	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication (typically Bearer token for Azure)
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
		return nil, fmt.Errorf("azure sentinel returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var sentinelResp SentinelTablesResponse
	if err := json.NewDecoder(resp.Body).Decode(&sentinelResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to DataSource objects
	dataSources := make([]types.DataSource, 0, len(sentinelResp.Value))
	for _, table := range sentinelResp.Value {
		ds := s.convertToDataSource(table, workspaceInfo["workspaceName"])
		dataSources = append(dataSources, ds)
	}

	return dataSources, nil
}

func (s *SentinelProvider) convertToDataSource(table struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Properties struct {
		RetentionInDays int    `json:"retentionInDays"`
		TotalRetention  int    `json:"totalRetentionInDays"`
		ArchiveRetention int   `json:"archiveRetentionInDays"`
		Plan            string `json:"plan"`
		Schema          struct {
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
			Description string `json:"description"`
			Columns     []struct {
				Name        string `json:"name"`
				Type        string `json:"type"`
				Description string `json:"description"`
			} `json:"columns"`
		} `json:"schema"`
	} `json:"properties"`
}, workspaceName string) types.DataSource {
	
	ds := types.DataSource{
		ID:          table.ID,
		Name:        table.Properties.Schema.Name,
		Title:       table.Properties.Schema.DisplayName,
		Type:        "log-analytics-table",
		Pattern:     table.Properties.Schema.Name,
		Description: table.Properties.Schema.Description,
		Status:      "active",
	}

	// Add tags based on table characteristics
	tags := []string{"azure", "sentinel"}
	if strings.HasSuffix(table.Properties.Schema.Name, "_CL") {
		tags = append(tags, "custom")
	}
	if table.Properties.Plan == "Analytics" {
		tags = append(tags, "analytics")
	}
	ds.Tags = tags

	// Add metadata
	ds.Metadata = map[string]interface{}{
		"workspace":         workspaceName,
		"retentionDays":     table.Properties.RetentionInDays,
		"totalRetention":    table.Properties.TotalRetention,
		"archiveRetention":  table.Properties.ArchiveRetention,
		"plan":              table.Properties.Plan,
		"columnCount":       len(table.Properties.Schema.Columns),
		"resourceId":        table.ID,
	}

	// Add column information
	if len(table.Properties.Schema.Columns) > 0 {
		columns := make([]map[string]string, len(table.Properties.Schema.Columns))
		for i, col := range table.Properties.Schema.Columns {
			columns[i] = map[string]string{
				"name":        col.Name,
				"type":        col.Type,
				"description": col.Description,
			}
		}
		ds.Metadata["columns"] = columns
	}

	return ds
}

func (s *SentinelProvider) addAuth(req *http.Request) {
	auth := s.config.Auth
	switch auth.Type {
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+auth.Token)
	default:
		// Azure typically uses Bearer tokens
		if auth.Token != "" {
			req.Header.Set("Authorization", "Bearer "+auth.Token)
		}
	}
}

func (s *SentinelProvider) ValidateConnection(ctx context.Context) error {
	workspaceInfo, err := s.parseWorkspaceFromEndpoint()
	if err != nil {
		return fmt.Errorf("invalid workspace endpoint: %w", err)
	}

	// Test connection by attempting to get workspace info
	apiURL := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.OperationalInsights/workspaces/%s",
		workspaceInfo["subscriptionId"],
		workspaceInfo["resourceGroupName"],
		workspaceInfo["workspaceName"])

	params := url.Values{}
	params.Add("api-version", "2022-10-01")
	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	if s.config.Auth != nil {
		s.addAuth(req)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Azure Sentinel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("azure sentinel health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (s *SentinelProvider) GetCapabilities() types.ProviderCapabilities {
	return types.ProviderCapabilities{
		SupportsRealTimeQueries: true,
		SupportsHistoricalData:  true,
		SupportedDataTypes:      []string{"log-analytics-table", "custom-table"},
		RequiresAuthentication:  true,
	}
}
