# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A fully-functional Go-based health monitoring application that periodically checks the availability and responsiveness of configured hosts. The application features a modern web dashboard with real-time status updates and latency visualization.

**Module**: `github.com/andrewsjg/healthchecker-claude`
**Go Version**: 1.24.0

### Key Features

- **Multiple Check Types**: Ping (ICMP) and HTTP/HTTPS health checks
- **Web Dashboard**: Real-time monitoring interface with auto-refresh (HTMX)
- **Latency Visualization**: Unicode sparkline charts showing latency trends over time
- **Dynamic Configuration**: Add, edit, and delete hosts via web UI
- **External Integration**: Optional healthcheck.io notifications
- **macOS Menu Bar**: System tray integration for background operation
- **Flexible Config**: Supports both YAML and TOML configuration formats

## Development Commands

### Setup
```bash
# Initialize the project and download dependencies
go mod tidy

# Download dependencies
go get
```

### Building
```bash
# Build the application
go build -o healthchecker ./cmd/healthchecker

# Build for specific platforms
GOOS=linux GOARCH=amd64 go build -o healthchecker-linux ./cmd/healthchecker
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run a specific test
go test -run TestName ./path/to/package

# Run tests with race detection
go test -race ./...
```

### Running
```bash
# Run the application directly
go run ./cmd/healthchecker

# Run with arguments
go run ./cmd/healthchecker [args]
```

### Code Quality
```bash
# Format code
go fmt ./...

# Run linter (requires golangci-lint)
golangci-lint run

# Vet code for suspicious constructs
go vet ./...
```

## Project Structure

```
healthchecker/
├── cmd/healthchecker/           # Main application entry point
│   └── main.go                  # Orchestrates checks and servers
├── internal/                    # Private application code
│   ├── checker/                 # Health check implementations
│   │   ├── checker.go          # Checker interface and registry
│   │   ├── ping.go             # ICMP ping checker
│   │   └── http.go             # HTTP/HTTPS checker
│   ├── config/                  # Configuration management
│   │   └── loader.go           # YAML/TOML parser
│   ├── healthcheckio/           # External notification client
│   │   └── client.go
│   ├── sparkline/               # Latency visualization
│   │   ├── sparkline.go        # Unicode sparkline generator
│   │   └── sparkline_test.go   # Sparkline tests
│   ├── web/                     # Web server and UI
│   │   ├── server.go           # HTTP handlers and routing
│   │   └── templates/          # HTML templates
│   │       ├── index.html
│   │       ├── hosts.html
│   │       └── host-form.html
│   └── systray/                 # macOS menu bar integration
├── pkg/models/                  # Shared data structures
│   └── config.go               # Configuration and result models
└── config.yaml                  # Runtime configuration
```

## Architecture Details

### Health Check Flow

1. **Initialization** (main.go):
   - Loads configuration from YAML/TOML
   - Registers available checkers (Ping, HTTP)
   - Starts web server
   - Launches health check loop

2. **Check Execution Loop**:
   - Runs checks at configured interval (e.g., 30s)
   - Executes checks in parallel for all hosts
   - Measures latency (time.Duration)
   - Updates results and latency history

3. **Data Storage**:
   - **Latest Results**: In-memory map per host/check
   - **Latency History**: Rolling buffer of last 50 measurements
   - Thread-safe access via sync.RWMutex

4. **Web Dashboard**:
   - Auto-refreshes every 5 seconds (HTMX)
   - Displays status, latency, and sparkline charts
   - Supports dynamic host/check management

### Latency Visualization

The application tracks latency variance using Unicode sparkline charts:
- Stores last **50 latency measurements** per host/check
- Generates sparklines from last **30 measurements**
- Characters: ▁▂▃▄▅▆▇█ (low to high)
- Auto-scales based on min/max values

**Sparkline Patterns**:
- Flat line (▃▃▃▃): Stable latency
- Ascending (▁▃▅█): Degrading performance
- Descending (█▅▃▁): Improving performance
- Spiky (▁▁█▁▁): Intermittent issues

### Checker Interface

All health checkers implement this interface:
```go
type Checker interface {
    Check(ctx context.Context, host Host, check Check) CheckResult
    Type() CheckType
}
```

New checker types can be added by:
1. Implementing the Checker interface
2. Registering in the checker registry
3. Adding configuration support

## Configuration

The application uses `config.yaml` (or `config.toml`) with the following structure:

```yaml
hosts:
  - name: Example Host
    address: example.com
    checks:
      - type: ping
        enabled: true
        timeout: 5s
      - type: http
        enabled: true
        timeout: 5s
        options:
          url: https://example.com
          expected_status: "200"
        healthcheck_io_url: https://hc-ping.com/your-uuid

check_interval: 30s
web_server_port: 8080
enable_console_log: true
```

### Configuration Options

- **check_interval**: How often to run health checks (e.g., "30s", "1m")
- **web_server_port**: Port for web dashboard (default: 8080)
- **enable_console_log**: Print check results to console (true/false)

### Check Types

**Ping (ICMP)**:
- Tests basic network connectivity
- Measures round-trip time (RTT)
- No additional options required

**HTTP/HTTPS**:
- Tests web service availability
- Options:
  - `url`: Full URL to check
  - `expected_status`: Expected HTTP status code (default: "200")

## Recent Updates

### Sparkline Visualization (Latest)
- Added Unicode sparkline charts for latency trends
- Implemented rolling history (50 measurements per check)
- Created sparkline generation utility with comprehensive tests
- Integrated into web dashboard with auto-scaling

### Current Status
- Fully functional health monitoring system
- Web dashboard with real-time updates
- Dynamic configuration management
- Production-ready with comprehensive test coverage
