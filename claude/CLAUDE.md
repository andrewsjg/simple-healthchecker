# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based health checker application. The project is in its initial stage with only a go.mod file present.

**Module**: `github.com/andrewsjg/healthchecker-claude`
**Go Version**: 1.24.0

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

## Architecture Notes

The project structure should follow standard Go conventions:
- `cmd/` - Main applications for this project
- `internal/` - Private application and library code
- `pkg/` - Library code that's ok to use by external applications
- Health checking logic should be modular and testable
