# Logging Quick Reference

## Setup

```go
import "universal-checker/internal/logger"

config := logger.LoggerConfig{
    Level:      logger.INFO,
    JSONFormat: false,
    OutputFile: "logs/app.log",
    BufferSize: 1000,
    Component:  "checker",
}

log, err := logger.NewStructuredLogger(config)
if err != nil {
    panic(err)
}
defer log.Close()
```

## Basic Logging

| Method | Usage | Example |
|--------|-------|---------|
| `Info()` | General information | `log.Info("Server started")` |
| `Debug()` | Debug details | `log.Debug("Processing item", fields)` |
| `Warn()` | Warnings | `log.Warn("Low memory", fields)` |
| `Error()` | Errors | `log.Error("Failed", err, fields)` |
| `Fatal()` | Fatal (exits) | `log.Fatal("Critical", err, fields)` |

## Specialized Methods

### Network Requests
```go
log.LogNetworkRequest(method, url, statusCode, latency, proxy, correlationID, err)
```

### Tasks
```go
log.LogTaskStart(taskID, taskType, correlationID)
log.LogTaskComplete(taskID, taskType, correlationID, duration, success, err)
```

### Proxies
```go
log.LogProxySelection(strategy, proxy, alternatives, correlationID)
log.LogHealthCheck(proxy, success, latency, err)
```

### Retries & Timeouts
```go
log.LogRetryAttempt(operation, attempt, maxAttempts, correlationID, lastError)
log.LogTimeout(operation, timeout, correlationID, proxy)
```

### Events
```go
log.LogCheckerEvent(eventType, result, fields)
log.LogProxyEvent(eventType, proxy, fields)
```

## Correlation Tracking

```go
correlationID := utils.GenerateCorrelationID()
log.LogWithCorrelation(logger.INFO, "Message", correlationID, fields)
```

## Fields

```go
fields := map[string]interface{}{
    "user_id": 123,
    "action":  "login",
    "ip":      "1.2.3.4",
}
log.Info("User action", fields)
```

## Log Levels

| Level | Value | When to Use |
|-------|-------|-------------|
| DEBUG | 0 | Development/troubleshooting |
| INFO | 1 | Normal operations |
| WARN | 2 | Potential issues |
| ERROR | 3 | Failures |
| FATAL | 4 | Critical failures (exits) |

## Output Formats

### Text (Readable)
```
[2025-11-01 18:45:04] INFO [checker] [CID:123] [Proxy:1.2.3.4:8080] [150ms] [HTTP:200] Message
```

### JSON (Machine-Parsable)
```json
{"time":"2025-11-01T18:45:04Z","level":"INFO","msg":"Message","correlation_id":"123"}
```

## Configuration Presets

```go
import "universal-checker/configs"

// Development
config := configs.DevelopmentLoggerConfig("checker")

// Production
config := configs.ProductionLoggerConfig("checker")

// Testing
config := configs.TestLoggerConfig("checker")
```

## Advanced

### Buffer Access
```go
recentLogs := log.GetRecentLogs(100)
```

### Export Logs
```go
log.ExportLogs("debug.json", 500)
```

### Dynamic Changes
```go
log.SetLevel(logger.DEBUG)
log.SetComponent("new-component")
```

## Performance Tips

1. Use INFO+ in production
2. Avoid logging in tight loops
3. Use correlation IDs for tracing
4. Prefer structured fields over string formatting
5. Always `defer log.Close()`

## Common Patterns

### Request Handler
```go
correlationID := utils.GenerateCorrelationID()
log.Info("Request received", map[string]interface{}{
    "correlation_id": correlationID,
    "method": req.Method,
    "path": req.URL.Path,
})

// ... process request ...

log.LogNetworkRequest(method, url, status, latency, proxy, correlationID, err)
```

### Worker Loop
```go
for task := range taskChan {
    taskID := utils.GenerateTaskID("work")
    correlationID := utils.GenerateCorrelationID()
    
    log.LogTaskStart(taskID, "process", correlationID)
    
    err := processTask(task)
    
    log.LogTaskComplete(taskID, "process", correlationID, time.Since(start), err == nil, err)
}
```

### Retry Logic
```go
for attempt := 1; attempt <= maxRetries; attempt++ {
    err := operation()
    if err == nil {
        break
    }
    
    if attempt < maxRetries {
        log.LogRetryAttempt("operation", attempt, maxRetries, correlationID, err)
        time.Sleep(backoff)
    }
}
```
