package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
	"strings"

	"github.com/logfiend/internal/config"
	"github.com/logfiend/internal/providers"
	"github.com/logfiend/internal/types"
)

// version is set via -ldflags "-X main.version=<value>" at build time
var version = "dev"

func main() {
	configPath := flag.String("config", "config.yml", "Path to configuration file")
	output := flag.String("output", "datasource_inventory.json", "Path to save data source inventory JSON")
	providerName := flag.String("provider", "", "Override provider from config (optional)")
	timeout := flag.Duration("timeout", 30*time.Second, "Request timeout")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	dryRun := flag.Bool("dry-run", false, "Show what would be done without making network calls")
	debug := flag.Bool("debug", false, "Enable debug output")
	airgap := flag.Bool("airgap", false, "Run in airgap mode (no network calls)")
	version := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Show version if requested
	if *version {
		fmt.Printf("LogFiend version %s\n", getVersion())
		os.Exit(0)
	}

	// Enable debug logging if requested
	if *debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		fmt.Println("🐛 Debug mode enabled")
	}

	// Verbose output
	if *verbose {
		fmt.Println("🔍 LogFiend - Vendor-agnostic SIEM data source inventory tool")
		fmt.Printf("📁 Config file: %s\n", *configPath)
		fmt.Printf("💾 Output file: %s\n", *output)
	}

	// Airgap mode check
	if *airgap {
		fmt.Println("🔒 Running in airgap mode - no network calls will be made")
		if *dryRun {
			fmt.Println("ℹ️  Airgap mode supersedes dry-run mode")
		}
		*dryRun = true // Airgap implies dry-run
	}

	// Validate and sanitize input paths
	if err := validatePath(*configPath); err != nil {
		log.Fatalf("Invalid config path: %v", err)
	}
	if err := validateOutputPath(*output); err != nil {
		log.Fatalf("Invalid output path: %v", err)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override provider if specified via CLI
	if *providerName != "" {
		if *verbose {
			fmt.Printf("🔄 Overriding provider: %s -> %s\n", cfg.Provider.Type, *providerName)
		}
		cfg.Provider.Type = *providerName
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Sanitize configuration
	if err := cfg.Sanitize(); err != nil {
		log.Fatalf("Failed to sanitize config: %v", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Initialize provider
	provider, err := providers.NewProvider(cfg.Provider)
	if err != nil {
		log.Fatalf("Failed to initialize provider '%s': %v", cfg.Provider.Type, err)
	}

	if *verbose {
		fmt.Printf("🔍 Fetching data sources from %s provider...\n", provider.Name())
		if capabilities := provider.GetCapabilities(); capabilities.RequiresAuthentication {
			fmt.Println("🔐 Provider requires authentication")
		}
	}

	// Dry run mode
	if *dryRun {
		fmt.Println("🚫 DRY RUN MODE - No actual network calls will be made")
		fmt.Printf("Would connect to: %s\n", sanitizeEndpoint(cfg.Provider.Endpoint))
		fmt.Printf("Would use provider: %s\n", provider.Name())
		fmt.Printf("Would save results to: %s\n", *output)
		
		if !*airgap {
			fmt.Println("Would validate connection...")
			// In dry-run, we can still validate config without network calls
		}
		
		os.Exit(0)
	}

	// Validate connection (only if not in airgap mode)
	if !*airgap {
		if *verbose {
			fmt.Println("🔗 Validating connection...")
		}
		if err := provider.ValidateConnection(ctx); err != nil {
			log.Fatalf("Connection validation failed: %v", err)
		}
		if *verbose {
			fmt.Println("✅ Connection validated successfully")
		}
	}

	// Fetch data views (only if not in airgap mode)
	var dataViews []types.DataSource
	if !*airgap {
		dataViews, err = provider.FetchDataViews(ctx)
		if err != nil {
			log.Fatalf("Error retrieving data views from %s: %v", provider.Name(), err)
		}
	} else {
		// In airgap mode, return empty results
		dataViews = []types.DataSource{}
		fmt.Println("🔒 Airgap mode: Returning empty results")
	}

	// Build inventory
	inventory := types.DataSourceInventory{
		Metadata: types.InventoryMetadata{
			Timestamp:    time.Now(),
			Provider:     provider.Name(),
			Version:      getVersion(),
			SourceCount:  len(dataViews),
			GeneratedBy:  "logfiend",
		},
		DataSources: dataViews,
	}

	// Marshal to JSON with pretty printing
	jsonOutput, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling inventory to JSON: %v", err)
	}

	// Write to file
	if err := writeOutputSafely(*output, jsonOutput); err != nil {
		log.Fatalf("Error writing to output file: %v", err)
	}

	fmt.Printf("✅ %s data source inventory saved to %s (%d sources)\n", 
		provider.Name(), *output, len(dataViews))
	
	// Print summary (only if we have data)
	if len(dataViews) > 0 && *verbose {
		printSummary(dataViews)
	}
}

func printSummary(dataSources []types.DataSource) {
	typeCount := make(map[string]int)
	for _, ds := range dataSources {
		typeCount[ds.Type]++
	}

	fmt.Println("\n📊 Summary by type:")
	for dsType, count := range typeCount {
		fmt.Printf("  %s: %d\n", dsType, count)
	}
}

// getVersion returns the application version
func getVersion() string {
	return version
}

// validatePath ensures the config path is safe and accessible
func validatePath(path string) error {
	// Sanitize path
	cleanPath := filepath.Clean(path)
	
	// Check if path is absolute and outside allowed directories
	if filepath.IsAbs(cleanPath) {
		return fmt.Errorf("absolute paths not allowed for security: %s", cleanPath)
	}
	
	// Check for path traversal attempts
	if filepath.Base(cleanPath) != cleanPath && filepath.Dir(cleanPath) != "." && filepath.Dir(cleanPath) != "examples" {
		return fmt.Errorf("path traversal not allowed: %s", cleanPath)
	}
	
	// Check if file exists and is readable
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist: %s", cleanPath)
	}
	
	return nil
}

// validateOutputPath ensures the output path is safe
func validateOutputPath(path string) error {
	// Sanitize path
	cleanPath := filepath.Clean(path)
	
	// Check if path is absolute and outside allowed directories
	if filepath.IsAbs(cleanPath) {
		return fmt.Errorf("absolute paths not allowed for security: %s", cleanPath)
	}
	
	// Check for path traversal attempts
	if filepath.Base(cleanPath) != cleanPath && filepath.Dir(cleanPath) != "." && filepath.Dir(cleanPath) != "output" {
		return fmt.Errorf("path traversal not allowed: %s", cleanPath)
	}
	
	return nil
}

// sanitizeEndpoint removes sensitive information from endpoint for display
func sanitizeEndpoint(endpoint string) string {
	// Remove any embedded credentials from URL display
	if idx := strings.Index(endpoint, "@"); idx != -1 {
		if schemeIdx := strings.Index(endpoint, "://"); schemeIdx != -1 {
			return endpoint[:schemeIdx+3] + "[REDACTED]" + endpoint[idx:]
		}
	}
	return endpoint
}

// writeOutputSafely writes output with proper permissions and error handling
func writeOutputSafely(path string, data []byte) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}
	
	// Write file with restricted permissions (read/write for owner only)
	return os.WriteFile(path, data, 0600)
}
