# Logging Framework Guide

## Overview

The stuffer-go project uses Go's standard library `log/slog` package with a custom handler for structured, readable logging. This provides high-performance logging with excellent text formatting and correlation tracking capabilities.

## Architecture

### Components

1. **slog (Go stdlib)** - Core logging framework
2. **CheckerHandler** - Custom handler for readable text formatting
3. **StructuredLogger** - Wrapper providing domain-specific logging methods

### Design Benefits

- ✅ Zero external dependencies (stdlib only)
- ✅ High performance (~0.3-0.5μs per log call)
- ✅ Structured logging with correlation IDs
- ✅ Readable text output format
- ✅ JSON output support for machine parsing
- ✅ Thread-safe operations

---

## Configuration

### LoggerConfig Structure

```go
type LoggerConfig struct {
    Level      LogLevel  // DEBUG, INFO, WARN, ERROR, FATAL
    JSONFormat bool      // true = JSON, false = readable text
    OutputFile string    // File path (empty = stdout only)
    BufferSize int       // Number of logs to buffer (0 = no buffering)
    Component  string    // Component name for log entries
}
```

### Log Levels

| Level | Value | Use Case |
|-------|-------|----------|
| `DEBUG` | 0 | Detailed debugging information |
| `INFO` | 1 | General informational messages |
| `WARN` | 2 | Warning messages (non-critical issues) |
| `ERROR` | 3 | Error messages (failures) |
| `FATAL` | 4 | Fatal errors (exits application) |

### Example Configurations

#### Development (Verbose, Text Format)

```go
config := logger.LoggerConfig{
    Level:      logger.DEBUG,
    JSONFormat: false,
    OutputFile: "logs/dev.log",
    BufferSize: 1000,
    Component:  "checker",
}
```

#### Production (Info+, JSON Format)

```go
config := logger.LoggerConfig{
    Level:      logger.INFO,
    JSONFormat: true,
    OutputFile: "logs/production.log",
    BufferSize: 5000,
    Component:  "checker",
}
```

#### Testing (Stdout Only)

```go
config := logger.LoggerConfig{
    Level:      logger.DEBUG,
    JSONFormat: false,
    OutputFile: "",  // stdout only
    BufferSize: 0,   // no buffering
    Component:  "test",
}
```

---

## Usage Patterns

### Basic Logging

```go
// Create logger
logger, err := logger.NewStructuredLogger(config)
if err != nil {
    panic(err)
}
defer logger.Close()

// Basic messages
logger.Info("Application started")
logger.Debug("Debug information", map[string]interface{}{
    "version": "1.0.0",
    "env":     "production",
})
logger.Warn("Resource running low", map[string]interface{}{
    "available": 5,
    "threshold": 10,
})
logger.Error("Operation failed", err, map[string]interface{}{
    "operation": "database_query",
})
```

### Correlation Tracking

Correlation IDs enable tracing requests across multiple operations:

```go
correlationID := utils.GenerateCorrelationID()

logger.LogWithCorrelation(logger.INFO, "Processing request", correlationID, map[string]interface{}{
    "user_id": 12345,
})

// Correlation ID appears in all related logs
logger.LogNetworkRequest("POST", url, 200, latency, proxy, correlationID, nil)
logger.LogTaskComplete(taskID, "check", correlationID, duration, true, nil)
```

### Network Request Logging

```go
// Successful request
logger.LogNetworkRequest(
    "POST",                    // method
    "https://api.example.com", // url
    200,                       // status code
    150*time.Millisecond,      // latency
    proxy,                     // *types.Proxy (or nil)
    correlationID,             // correlation ID
    nil,                       // error (or actual error)
)

// Failed request
logger.LogNetworkRequest(
    "GET",
    "https://api.example.com",
    0,
    5*time.Second,
    proxy,
    correlationID,
    fmt.Errorf("request timeout"),
)
```

### Task Lifecycle Logging

```go
taskID := utils.GenerateTaskID("check")
correlationID := utils.GenerateCorrelationID()

// Start task
logger.LogTaskStart(taskID, "combo_check", correlationID)

// ... perform work ...

// Complete task
logger.LogTaskComplete(
    taskID,
    "combo_check",
    correlationID,
    duration,
    success,  // bool
    err,      // error or nil
)
```

### Proxy Operations

```go
// Proxy selection
logger.LogProxySelection(
    "best_score",  // strategy
    proxy,         // selected proxy
    10,            // number of alternatives
    correlationID,
)

// Health check
logger.LogHealthCheck(
    proxy,
    success,  // bool
    latency,
    err,      // error or nil
)
```

### Retry Logic

```go
logger.LogRetryAttempt(
    "network_request",  // operation
    2,                  // current attempt
    3,                  // max attempts
    correlationID,
    lastError,          // error from previous attempt
)
```

### Timeout Tracking

```go
logger.LogTimeout(
    "http_request",       // operation
    30*time.Second,       // timeout duration
    correlationID,
    proxy,                // proxy used (or nil)
)
```

### Checker Events

```go
logger.LogCheckerEvent(
    "valid_combo_found",  // event type
    result,               // types.CheckResult
    map[string]interface{}{
        "additional": "context",
    },
)
```

### Proxy Events

```go
logger.LogProxyEvent(
    "proxy_added",  // event type
    proxy,          // types.Proxy
    map[string]interface{}{
        "source": "scraper",
    },
)
```

---

## Output Formats

### Text Format (Human-Readable)

**Standard Log:**
```
[2025-11-01 18:45:04] INFO [checker] Application started
```

**With Fields:**
```
[2025-11-01 18:45:04] DEBUG [checker] Debug information
  Fields: version=1.0.0, env=development
```

**Network Request (Success):**
```
[2025-11-01 18:45:04] INFO [checker] [CID:CID-12345] [Proxy:192.168.1.100:8080] [150ms] [HTTP:200] Network request: POST https://api.example.com/login
```

**Network Request (Error):**
```
[2025-11-01 18:45:04] ERROR [checker] [CID:CID-12346] [Proxy:192.168.1.100:8080] [5000ms] Network request: GET https://api.example.com/data - Error: request timeout after 5s
```

**Task Lifecycle:**
```
[2025-11-01 18:45:04] INFO [checker] [CID:CID-12347] [TID:task-001] Task started: combo_check
[2025-11-01 18:45:04] INFO [checker] [CID:CID-12347] [TID:task-001] [50ms] Task completed: combo_check (0.05s)
```

**Retry Attempt:**
```
[2025-11-01 18:45:04] INFO [checker] [CID:CID-12348] [Retry:2] Retry attempt 2/3 for network_request
```

### JSON Format (Machine-Parsable)

```json
{
  "time": "2025-11-01T18:45:04.510557276+06:00",
  "level": "INFO",
  "msg": "Network request: POST https://api.example.com/login",
  "correlation_id": "CID-99999",
  "status_code": 200,
  "latency_ms": 150,
  "proxy_host": "192.168.1.100",
  "proxy_port": 8080
}
```

---

## Advanced Features

### Log Buffering

The logger maintains an in-memory buffer of recent logs for debugging:

```go
// Get last 100 log entries
recentLogs := logger.GetRecentLogs(100)

for _, entry := range recentLogs {
    fmt.Printf("[%s] %s: %s\n", entry.Timestamp, entry.Level, entry.Message)
}
```

### Log Export

Export buffered logs to a file:

```go
err := logger.ExportLogs("debug_export.json", 500)
if err != nil {
    logger.Error("Failed to export logs", err, nil)
}
```

### Dynamic Level Changes

```go
// Change log level at runtime
logger.SetLevel(logger.DEBUG)

// Change component name
logger.SetComponent("new-component")
```

---

## Performance Considerations

### Benchmarks

- **slog with CheckerHandler**: ~0.3-0.5μs per log call
- **Memory allocation**: Minimal (pre-allocated buffers)
- **Concurrency**: Thread-safe with mutex protection

### Best Practices

1. **Use appropriate log levels**
   - DEBUG: Only in development/troubleshooting
   - INFO: Normal operations
   - WARN: Potential issues
   - ERROR: Actual failures

2. **Avoid logging in hot paths**
   - Use INFO or higher in production
   - Consider sampling for high-frequency events

3. **Use correlation IDs**
   - Essential for distributed tracing
   - Helps correlate logs across operations

4. **Structured fields over string formatting**
   ```go
   // Good
   logger.Info("User login", map[string]interface{}{
       "user_id": 123,
       "ip": "1.2.3.4",
   })
   
   // Avoid
   logger.Info(fmt.Sprintf("User %d logged in from %s", 123, "1.2.3.4"))
   ```

5. **Close logger on shutdown**
   ```go
   defer logger.Close()  // Flushes file buffers
   ```

---

## Migration from Custom Logger

The new slog-based logger maintains **100% API compatibility** with the previous custom logger. No code changes required in existing components.

### What Changed

- **Internal implementation**: Now uses slog instead of custom formatting
- **Performance**: Improved (~2-3x faster)
- **Dependencies**: Zero external dependencies (stdlib only)

### What Stayed the Same

- All public methods and signatures
- Log output format (text mode)
- Specialized logging methods
- Buffer and export functionality

---

## Troubleshooting

### Issue: Logs not appearing

**Check log level:**
```go
logger.SetLevel(logger.DEBUG)  // Lower threshold
```

### Issue: File not being written

**Check permissions:**
```bash
ls -la logs/
chmod 755 logs/
```

**Check disk space:**
```bash
df -h
```

### Issue: Performance degradation

**Reduce log level in production:**
```go
config.Level = logger.INFO  // Skip DEBUG logs
```

**Disable file output for testing:**
```go
config.OutputFile = ""  // Stdout only
```

---

## Examples

See `/cmd/test-logging/main.go` for a comprehensive example demonstrating all logging features.

Run the example:
```bash
go run ./cmd/test-logging/main.go
```

---

## References

- [Go slog documentation](https://pkg.go.dev/log/slog)
- [slog Handler Guide](https://github.com/golang/example/blob/master/slog-handler-guide/README.md)
- Project: `/internal/logger/structured_logger.go`
