## Architecture

This project is a Go CLI that inventories data sources from multiple SIEM platforms.

### Core Flow
1. Parse flags in `main.go`
2. Load and sanitize config via `internal/config`
3. Construct provider via `internal/providers.NewProvider`
4. Optionally validate connection
5. Fetch data views
6. Serialize inventory to JSON and write safely

### Packages
- `internal/types`
  - `Provider` interface and `ProviderCapabilities`
  - `DataSource`, `DataSourceInventory`, `InventoryMetadata`
  - `ProviderConfig`, `AuthConfig`, `TLSConfig`
- `internal/config`
  - `Load(path)` reads YAML into `Config`
  - `Validate()` ensures required fields
  - `Sanitize()` normalizes values and enforces safe endpoints
- `internal/providers`
  - Registry (`Register`, `NewProvider`, `GetAvailableProviders`)
  - Built-ins: elasticsearch, splunk, sentinel, qradar

### CLI Flags
- Security: `--dry-run`, `--airgap`, `--debug`, `--verbose`
- IO: `--config`, `--output`, `--timeout`
- Misc: `--provider`, `--version`

### Versioning
`version` is injected at build time using `-ldflags "-X main.version=<value>"`.

### Security Considerations
- No absolute paths for config/output; prevents path traversal
- Credentials are never logged
- HTTP allowed only for localhost; otherwise HTTPS is required
- Output files are written with `0600` permissions

### Extensibility
To add a provider, implement `types.Provider` and register it in `providers.go`. 