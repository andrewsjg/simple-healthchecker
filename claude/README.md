# Health Checker

A simple, efficient health monitoring application written in Go that checks the liveness of configured hosts and provides a web-based dashboard for monitoring and management.

## Features

- **Multiple Configuration Formats**: Supports both YAML and TOML configuration files
- **Ping Health Checks**: ICMP ping checks for host availability (extensible for HTTP checks and more)
- **Web Dashboard**: Interactive HTMX-based web UI for real-time monitoring
- **Enable/Disable Checks**: Toggle individual health checks on/off via the web interface
- **Healthcheck.io Integration**: Optional integration with healthcheck.io for notifications
- **Auto-refresh**: Dashboard automatically refreshes every 5 seconds
- **Configurable Intervals**: Set custom check intervals and timeouts
- **Graceful Shutdown**: Proper signal handling for clean shutdowns

## Installation

### Prerequisites

- Go 1.24.0 or higher

### Build from Source

```bash
git clone https://github.com/andrewsjg/healthchecker-claude.git
cd healthchecker-claude
go build -o healthchecker ./cmd/healthchecker
```

## Configuration

Create a configuration file in either YAML or TOML format. Example configurations are provided:

### YAML Configuration (config.yaml)

```yaml
check_interval: 60s
web_server_port: 8080

hosts:
  - name: "Google DNS"
    address: "8.8.8.8"
    checks:
      - type: "ping"
        enabled: true
        timeout: 5s
        healthcheck_io_url: "https://hc-ping.com/your-uuid-1"

  - name: "Cloudflare DNS"
    address: "1.1.1.1"
    checks:
      - type: "ping"
        enabled: true
        timeout: 5s
        healthcheck_io_url: "https://hc-ping.com/your-uuid-2"
```

### TOML Configuration (config.toml)

```toml
check_interval = "60s"
web_server_port = 8080

[[hosts]]
name = "Google DNS"
address = "8.8.8.8"

[[hosts.checks]]
type = "ping"
enabled = true
timeout = "5s"
healthcheck_io_url = "https://hc-ping.com/your-uuid-1"
```

### Configuration Options

- `check_interval`: How often to run health checks (e.g., "60s", "5m")
- `web_server_port`: Port for the web dashboard (default: 8080)
- `hosts`: List of hosts to monitor
  - `name`: Display name for the host
  - `address`: IP address or hostname
  - `checks`: List of health checks for this host
    - `type`: Type of check ("ping" currently supported)
    - `enabled`: Whether the check is active
    - `timeout`: Maximum time to wait for a response
    - `healthcheck_io_url`: (Optional) Unique healthcheck.io ping URL for this specific check

## Usage

### Running the Application

```bash
# Use default config file (config.yaml)
./healthchecker

# Specify a custom config file
./healthchecker -config /path/to/config.yaml

# Using TOML configuration
./healthchecker -config config.toml
```

### Accessing the Web Dashboard

Once running, open your browser to:

```
http://localhost:8080
```

The dashboard shows:
- All configured hosts
- Status of each health check (success/failure)
- Response times
- Enable/disable buttons for each check
- Auto-refreshing status every 5 seconds

## Project Structure

```
healthchecker-claude/
├── cmd/
│   └── healthchecker/        # Main application entry point
│       └── main.go
├── internal/
│   ├── checker/              # Health check implementations
│   │   ├── checker.go        # Checker interface and registry
│   │   └── ping.go           # Ping checker implementation
│   ├── config/               # Configuration loading
│   │   ├── loader.go
│   │   └── loader_test.go
│   ├── healthcheckio/        # Healthcheck.io integration
│   │   └── client.go
│   └── web/                  # Web server and UI
│       ├── server.go
│       └── templates/
│           ├── index.html
│           └── hosts.html
├── pkg/
│   └── models/               # Shared data models
│       └── config.go
├── config.example.yaml       # Example YAML configuration
├── config.example.toml       # Example TOML configuration
├── go.mod
└── README.md
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...
```

### Adding New Check Types

To add a new check type (e.g., HTTP):

1. Add the new check type to `pkg/models/config.go`:
   ```go
   const (
       CheckTypePing CheckType = "ping"
       CheckTypeHTTP CheckType = "http"
   )
   ```

2. Create a new checker in `internal/checker/`:
   ```go
   type HTTPChecker struct{}

   func (h *HTTPChecker) Type() models.CheckType {
       return models.CheckTypeHTTP
   }

   func (h *HTTPChecker) Check(ctx context.Context, host models.Host, check models.Check) models.CheckResult {
       // Implement HTTP check logic
   }
   ```

3. Register the checker in `cmd/healthchecker/main.go`:
   ```go
   registry.Register(checker.NewHTTPChecker())
   ```

## Healthcheck.io Integration

To enable healthcheck.io notifications for individual checks:

1. Sign up at [healthcheck.io](https://healthcheck.io)
2. Create a new check for each health check you want to monitor and copy each unique ping URL
3. Add the URLs to your configuration file for each check:
   ```yaml
   hosts:
     - name: "My Server"
       address: "example.com"
       checks:
         - type: "ping"
           enabled: true
           timeout: 5s
           healthcheck_io_url: "https://hc-ping.com/your-uuid-here"
   ```

The application will:
- Send a success signal to the specific healthcheck.io monitor when that check passes
- Send a failure signal to the specific healthcheck.io monitor when that check fails
- Only send notifications for checks that have a `healthcheck_io_url` configured
- Allow you to have different notification settings for different checks

## Roadmap

- [ ] HTTP/HTTPS health checks
- [ ] TCP port checks
- [ ] Custom check scripts
- [ ] Email notifications
- [ ] Slack/Discord webhooks
- [ ] Check history and graphs
- [ ] Docker image
- [ ] Persistent state storage

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Built with [go-ping](https://github.com/go-ping/ping) for ICMP ping functionality
- Web UI powered by [HTMX](https://htmx.org)
- Configuration parsing with [go-yaml](https://github.com/go-yaml/yaml) and [go-toml](https://github.com/pelletier/go-toml)
