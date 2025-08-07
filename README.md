## Installation

### Download Binary
```bash
# Download the latest release
curl -L https://github.com/logfiend/logfiend/releases/latest/download/logfiend-linux-amd64 -o logfiend
chmod +x logfiend
```

### Build from Source
```bash
# Clone the repository
git clone https://github.com/logfiend/logfiend.git
cd logfiend

# Build the application
make build

# The binary will be in ./build/logfiend
```

### Verification
```bash
# Verify the installation
./logfiend --version

# Check the binary hash (hashes provided with each release)
sha256sum logfiend
```

### Uninstall
LogFiend respects your system - to uninstall, simply delete the binary:
```bash
rm /path/to/logfiend
```

## Quick Start

### 1. Create Configuration

Create a `config.yml` file (never include credentials directly):

```yaml
provider:
  type: "elasticsearch"
  endpoint: "https://localhost:9200"
  auth:
    type: "basic"
    username: "${ELASTIC_USERNAME}"  # Read from environment
    password: "${ELASTIC_PASSWORD}"  # Read from environment
```

### 2. Set Environment Variables

```bash
export ELASTIC_USERNAME="elastic"
export ELASTIC_PASSWORD="your-password"
```

### 3. Test Configuration (Dry Run)

```bash
# Test without making network calls
./logfiend --config=config.yml --dry-run

# Test in airgap mode (no network access)
./logfiend --config=config.yml --airgap

# Verbose output for debugging
./logfiend --config=config.yml --dry-run --verbose
```

### 4. Run Inventory

```bash
# Run the inventory
./logfiend --config=config.yml --output=inventory.json

# With verbose output
./logfiend --config=config.yml --output=inventory.json --verbose
```

### 5. View Results

```bash
# View metadata
cat inventory.json | jq '.metadata'

# Count data sources by type
cat inventory.json | jq '.data_sources | group_by(.type) | map({type: .[0].type, count: length})'
```

## Command Line Options

```bash
Usage: logfiend [OPTIONS]

Security & Behavior:
  --dry-run         Show what would be done without making network calls
  --airgap          Run in airgap mode (no network calls)
  --verbose         Enable verbose output
  --debug           Enable debug output
  --version         Show version information

Configuration:
  --config string   Path to configuration file (default "config.yml")
  --provider string Override provider from config (optional)
  --timeout duration Request timeout (default 30s)

Output:
  --output string   Path to save inventory JSON (default "datasource_inventory.json")
```

## Example Commands

```bash
# Basic inventory with verbose output
./logfiend --config=config.yml --verbose

# Test configuration without network calls
./logfiend --config=config.yml --dry-run

# Run in airgap mode (no network access)
./logfiend --config=config.yml --airgap

# Override provider from command line
./logfiend --config=config.yml --provider=splunk

# Custom timeout and output location
./logfiend --config=config.yml --timeout=60s --output=./reports/inventory.json

# Debug mode for troubleshooting
./logfiend --config=config.yml --debug --verbose
```

## Example Output

```json
{
  "metadata": {
    "timestamp": "2024-01-15T10:30:00Z",
    "provider": "elasticsearch",
    "version": "1.0.0",
    "source_count": 42,
    "generated_by": "logfiend"
  },
  "data_sources": [
    {
      "id": "logs-nginx-*",
      "name": "logs-nginx-*",
      "title": "Nginx Logs",
      "type": "index-pattern",
      "pattern": "logs-nginx-*",
      "description": "Nginx access and error logs",
      "status": "active",
      "tags": ["web", "nginx"],
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-15T09:00:00Z",
      "metadata": {
        "timeField": "@timestamp",
        "fieldCount": 25
      }
    }
  ]
}
```

## Screenshots

### Dry Run Mode
```
🚫 DRY RUN MODE - No actual network calls will be made
Would connect to: https://[REDACTED]@localhost:9200
Would use provider: elasticsearch
Would save results to: inventory.json
```

---

## AI Maintenance
- See `docs/ARCHITECTURE.md` for structure and flow
- Use `cursorrules` for assistant guidance
- Lint: `golangci-lint run` (CI enforces it)
- Test: `make test`
- Build: `make build` (version injected via ldflags)
- Provider contracts live in `internal/types`; register in `internal/providers/providers.go`
