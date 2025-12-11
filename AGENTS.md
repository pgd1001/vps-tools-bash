# AGENTS.md

## Build Commands
- `make build` - Build binary to bin/vps-tools
- `make test` - Run all tests  
- `go test ./path/to/package -run TestSpecific` - Run single test
- `make fmt` - Format code with gofmt
- `make lint` - Run golangci-lint
- `make tui` - Run interactive TUI mode
- `make run` - Run CLI mode

## Code Style Guidelines
- **Formatting**: Use `gofmt` and `golangci-lint` (enforced in CI)
- **Imports**: Group imports in three blocks (standard, third-party, internal)
- **Naming**: CamelCase for exported, camelCase for unexported
- **Types**: Use strong typing with structs, avoid interface{} unless necessary
- **Error Handling**: Always handle errors explicitly, use fmt.Errorf for wrapping
- **SSH**: Prefer `golang.org/x/crypto/ssh` library over shelling out to ssh command
- **Security**: Never store plaintext secrets, use SSH agent or OS keyring
- **Testing**: Unit tests for core logic, integration tests for SSH wrapper
- **Architecture**: Follow package structure in architecture.md (cmd/, internal/, pkg/)

## Key Libraries
- Bubble Tea for TUI architecture
- Cobra for CLI commands  
- BoltDB/SQLite for local storage
- golang.org/x/crypto/ssh for SSH operations