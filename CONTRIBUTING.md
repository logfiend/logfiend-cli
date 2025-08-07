## Contributing

Thank you for considering a contribution. This guide is optimized for both humans and AI assistants working in tools like Cursor.

### Quick Start
- Fork and clone the repo
- Install Go 1.21+
- Install deps: `make deps`
- Run tests: `make test`
- Build: `make build`

### Code Style
- Follow idiomatic Go and the existing patterns in `internal/`
- Use descriptive names; avoid abbreviations
- Prefer early returns and clear error handling
- Keep functions small and focused

### Linting
- Lint locally with `golangci-lint run` or rely on CI
- The configuration is in `.golangci.yml`

### Tests
- Add unit tests for new logic
- Focus on edge cases and failure paths
- Keep tests deterministic; avoid network I/O

### Architecture
- `main.go` handles CLI, flags, IO, and orchestration
- `internal/config` loads, validates, and sanitizes config
- `internal/types` defines core types and the `Provider` interface
- `internal/providers` implements provider registry and providers

### Adding Providers
1. Create a new file in `internal/providers`
2. Implement `types.Provider`
3. Register it in `internal/providers/providers.go`

### Security
- Never log secrets
- Prefer HTTPS; only allow HTTP for localhost per `config` sanitization
- Respect `--airgap` and `--dry-run`

### Commits and PRs
- Keep PRs small and focused
- Reference issues when relevant
- Describe behavior changes and risks

### Local Development Commands
- `make deps` — install modules
- `make test` — run tests
- `make build` — build binary to `./build`
- `make run` — run with `config.yml`

### Release
- Version is injected via ldflags; see `Makefile` 