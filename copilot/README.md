# Simple Healthchecker

A small Go service that periodically runs health checks (Ping and HTTP) against a list of hosts from a YAML/TOML config, shows live status in a built-in web UI (HTMX), and optionally notifies Healthchecks.io.

## Features
- Hosts defined in config with one or more checks per host
- Checks: ping, http (with expected status code)
- Enable/disable checks per host
- Web UI to add/edit/delete hosts and add/remove/update checks
- “Unknown” status until a host’s checks run the first time
- Optional Healthchecks.io ping URL per host for notifications
- Live UI updates without manual refresh

## Build
- go build ./cmd/simple-healthchecker
- Binary: ./simple-healthchecker

## Run
- ./simple-healthchecker -config config.yaml -addr :8080
- Open http://localhost:8080

## Command-line options
- -config string      Path to config file (YAML or TOML). Default: config.yaml
- -addr string        HTTP listen address. Default: :8080
- -interval duration  Check interval (e.g. 30s, 1m). Default: 30s
- -log string         Path to log file (optional; defaults to stderr)
- -http-log           Enable web server request logging (disabled by default)

On start, the app logs: “simple-healthchecker started; web UI listening on <addr>”.

## Configuration
YAML example (see config.example.yaml for a fuller sample):

```yaml
hosts:
  - name: "router"
    address: "192.168.1.1"
    healthchecks_ping_url: "https://hc-ping.com/<uuid>"   # optional
    checks:
      - type: ping
        enabled: true
      - type: http
        url: "https://example.com/health"
        expect: 200         # optional; defaults to 200 when omitted
        enabled: true
  - name: "api"
    address: "api.internal"
    checks:
      - type: http
        url: "http://api.internal/ready"
        expect: 204
        enabled: true
```

Notes:
- type: ping has no URL or expect; just enabled flag.
- type: http requires url; expect is optional (defaults to 200).
- healthchecks_ping_url is optional per host. If set, failures will be reported and recoveries can be marked OK.

TOML uses equivalent keys.

## Web UI
- Cards show each host, its checks, last status (UP/DOWN/UNKNOWN), latency, and last-checked time.
- Edit dialog lets you:
  - Change host name/address and Healthchecks.io URL
  - Add new checks (Ping/HTTP) and remove existing checks
  - For HTTP checks: set target URL and expected status code
- Main view keeps card order stable and auto-refreshes periodically.

## Healthchecks.io integration
- Set healthchecks_ping_url on a host to enable notifications.
- The service will call Healthchecks.io endpoints based on check outcomes.

## Logging
- By default logs to stderr; use -log /path/app.log to write to a file.
- Use -http-log to add request logs for the web UI endpoints.

## ICMP on macOS
- Standard raw-ICMP requires privileges on macOS. This app includes a Darwin-specific option to perform ping checks without requiring root. If ping checks fail due to permissions, ensure you’re on the latest build of this app.
