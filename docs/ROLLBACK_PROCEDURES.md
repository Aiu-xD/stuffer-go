# Rollback Procedures - Go Version Upgrade and Dependency Management

## Overview
This document provides rollback procedures for Task 1.1 - Go Version Upgrade and Dependency Management changes.

## Changes Made
- **Go Version**: Upgraded from `1.24.0` to `1.23`
- **Dependencies**: Updated to latest compatible versions
- **azuretls-client**: Maintained at v1.12.9 (compatible)

## Rollback Steps

### 1. Restore Previous Go Version
```bash
# Restore go.mod Go version
sed -i 's/go 1.23/go 1.24.0/' go.mod
```

### 2. Restore Previous Dependencies
```bash
# Option A: Use git to restore go.mod and go.sum
git checkout HEAD~1 -- go.mod go.sum

# Option B: Manual restoration (if git not available)
# Restore key dependencies to previous versions:
go mod edit -require=github.com/spf13/cobra@v1.7.0
go mod edit -require=github.com/PuerkitoBio/goquery@v1.8.1
go mod edit -require=github.com/klauspost/compress@v1.18.0
go mod edit -require=golang.org/x/tools@v0.37.0
go mod edit -require=golang.org/x/mod@v0.28.0
```

### 3. Clean and Verify
```bash
# Clean module cache and tidy
go clean -modcache
go mod tidy
go mod download

# Verify builds
go build -o /tmp/test ./cmd/main.go
go build -o /tmp/test-gui ./cmd/gui/main.go
```

### 4. Validation Tests
```bash
# Run existing tests
go test ./...

# Verify performance (if benchmarks available)
go test -bench=. ./...
```

## Critical Dependencies to Monitor
- **azuretls-client**: Must remain compatible (currently v1.12.9)
- **Fyne GUI**: v2.7.0 (critical for GUI functionality)
- **Cobra CLI**: For command-line interface
- **goquery**: For HTML parsing functionality

## Emergency Contacts
- Backup go.mod and go.sum files are available in git history
- Performance baseline: 1,365+ CPM must be maintained
- All builds must pass without errors

## Verification Checklist
- [ ] Go version matches expected (1.24.0 for rollback)
- [ ] All dependencies resolve without conflicts
- [ ] CLI build succeeds: `go build ./cmd/main.go`
- [ ] GUI build succeeds: `go build ./cmd/gui/main.go`
- [ ] Test suite passes: `go test ./...`
- [ ] Performance benchmarks meet baseline
- [ ] azuretls-client functionality preserved

## Notes
- Always test rollback in development environment first
- Monitor for any breaking changes in dependency behavior
- Document any issues encountered during rollback process
