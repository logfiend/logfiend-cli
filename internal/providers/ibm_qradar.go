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

// QRadarProvider implements the Provider interface for IBM QRadar
type QRadarProvider struct {
	config types.ProviderConfig
	client *http.Client
}

// QRadarLogSource represents a QRadar log source
type QRadarLogSource struct {
	ID                  int    `json:"id"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	TypeID              int    `json:"type_id"`
	ProtocolTypeID      int    `json:"protocol_type_id"`
	Enabled             bool   `json:"enabled"`
	Gateway             bool   `json:"gateway"`
	Internal            bool   `json:"internal"`
	Credibility         int    `json:"credibility"`
	TargetEventRate     int    `json:"target_event_rate"`
	LogSourceExtension  interface{} `json:"log_source_extension"`
	CreationDate        int64  `json:"creation_date"`
	ModifiedDate        int64  `json:"modified_date"`
	LastEventTime       int64  `json:"last_event_time"`
	Status              struct {
		LastSeen int64  `json:"last_seen"`
		Messages []string `json:"messages"`
	} `json:"status"`
	AutoDiscovered      bool   `json:"auto_discovered"`
	AverageEPS          int    `json:"average_eps"`
}

// NewQRadarProvider creates a new QRadar provider
func NewQRadarProvider(config types.ProviderConfig) (types.Provider, error) {
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

	return &QRadarProvider{
		config: config,
		client: client,
	}, nil
}

func (q *QRadarProvider) Name() string {
	return "qradar"
}

func (q *QRadarProvider) FetchDataViews(ctx context.Context) ([]types.DataSource, error) {
	return q.fetchLogSources(ctx)
}

func (q *QRadarProvider) fetchLogSources(ctx context.Context) ([]types.DataSource, error) {
	// Build URL for QRadar's log sources API
	baseURL := strings.TrimSuffix(q.config.Endpoint, "/")
	endpoint := fmt.Sprintf("%s/api/config/event_sources/log_source_management/log_sources", baseURL)
	
	// Add query parameters
	params := url.Values{}
	params.Add("fields", "id,name,description,type_id,protocol_type_id,enabled,gateway,internal,credibility,target_event_rate,creation_date,modified_date,last_event_time,status,auto_discovered,average_eps")
	
	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Version", "15.0") // QRadar API version

	// Add authentication
	if q.config.Auth != nil {
		q.addAuth(req)
	}

	// Execute request
	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qradar returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var logSources []QRadarLogSource
	if err := json.NewDecoder(resp.Body).Decode(&logSources); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to DataSource objects
	dataSources := make([]types.DataSource, 0, len(logSources))
	for _, logSource := range logSources {
		ds := q.convertToDataSource(logSource)
		dataSources = append(dataSources, ds)
	}

	return dataSources, nil
}

func (q *QRadarProvider) convertToDataSource(logSource QRadarLogSource) types.DataSource {
	ds := types.DataSource{
		ID:          fmt.Sprintf("%d", logSource.ID),
		Name:        logSource.Name,
		Title:       logSource.Name,
		Type:        "qradar-log-source",
		Description: logSource.Description,
	}

	// Determine status
	if logSource.Enabled {
		ds.Status = "enabled"
	} else {
		ds.Status = "disabled"
	}

	// Add tags based on log source characteristics
	tags := []string{"qradar"}
	if logSource.Internal {
		tags = append(tags, "internal")
	} else {
		tags = append(tags, "external")
	}
	if logSource.Gateway {
		tags = append(tags, "gateway")
	}
	if logSource.AutoDiscovered {
		tags = append(tags, "auto-discovered")
	}
	ds.Tags = tags

	// Convert timestamps
	if logSource.CreationDate > 0 {
		createdAt := time.Unix(logSource.CreationDate/1000, 0)
		ds.CreatedAt = &createdAt
	}
	if logSource.ModifiedDate > 0 {
		updatedAt := time.Unix(logSource.ModifiedDate/1000, 0)
		ds.UpdatedAt = &updatedAt
	}

	// Add metadata
	ds.Metadata = map[string]interface{}{
		"typeId":           logSource.TypeID,
		"protocolTypeId":   logSource.ProtocolTypeID,
		"credibility":      logSource.Credibility,
		"targetEventRate":  logSource.TargetEventRate,
		"averageEPS":       logSource.AverageEPS,
		"gateway":          logSource.Gateway,
		"internal":         logSource.Internal,
		"autoDiscovered":   logSource.AutoDiscovered,
	}

	// Add last event time if available
	if logSource.LastEventTime > 0 {
		lastEvent := time.Unix(logSource.LastEventTime/1000, 0)
		ds.Metadata["lastEventTime"] = lastEvent.Format(time.RFC3339)
	}

	// Add status information
	if logSource.Status.LastSeen > 0 {
		lastSeen := time.Unix(logSource.Status.LastSeen/1000, 0)
		ds.Metadata["lastSeen"] = lastSeen.Format(time.RFC3339)
	}
	if len(logSource.Status.Messages) > 0 {
		ds.Metadata["statusMessages"] = logSource.Status.Messages
	}

	return ds
}

func (q *QRadarProvider) addAuth(req *http.Request) {
	auth := q.config.Auth
	switch auth.Type {
	case "basic":
		req.SetBasicAuth(auth.Username, auth.Password)
	case "api_key":
		req.Header.Set("SEC", auth.APIKey) // QRadar uses SEC header for API keys
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+auth.Token)
	}
}

func (q *QRadarProvider) ValidateConnection(ctx context.Context) error {
	baseURL := strings.TrimSuffix(q.config.Endpoint, "/")
	url := fmt.Sprintf("%s/api/system/about", baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	req.Header.Set("Version", "15.0")

	if q.config.Auth != nil {
		q.addAuth(req)
	}

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to QRadar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qradar health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (q *QRadarProvider) GetCapabilities() types.ProviderCapabilities {
	return types.ProviderCapabilities{
		SupportsRealTimeQueries: true,
		SupportsHistoricalData:  true,
		SupportedDataTypes:      []string{"qradar-log-source", "qradar-flow-source"},
		RequiresAuthentication:  true,
	}
}
