package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"universal-checker/pkg/types"
)

// LogLevel represents the level of logging
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of LogLevel
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// toSlogLevel converts LogLevel to slog.Level
func (l LogLevel) toSlogLevel() slog.Level {
	switch l {
	case DEBUG:
		return slog.LevelDebug
	case INFO:
		return slog.LevelInfo
	case WARN:
		return slog.LevelWarn
	case ERROR:
		return slog.LevelError
	case FATAL:
		return slog.LevelError + 1 // Fatal is higher than Error
	default:
		return slog.LevelInfo
	}
}

// LogEntry represents a structured log entry (for buffering and export)
type LogEntry struct {
	Timestamp     time.Time              `json:"timestamp"`
	Level         string                 `json:"level"`
	Message       string                 `json:"message"`
	Component     string                 `json:"component,omitempty"`
	Session       string                 `json:"session,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	TaskID        string                 `json:"task_id,omitempty"`
	ProxyHost     string                 `json:"proxy_host,omitempty"`
	ProxyPort     int                    `json:"proxy_port,omitempty"`
	Latency       int                    `json:"latency,omitempty"`
	StatusCode    int                    `json:"status_code,omitempty"`
	RetryAttempt  int                    `json:"retry_attempt,omitempty"`
	Timeout       time.Duration          `json:"timeout,omitempty"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
	Error         string                 `json:"error,omitempty"`
}

// CheckerHandler is a custom slog.Handler for readable text formatting
type CheckerHandler struct {
	opts      *slog.HandlerOptions
	output    io.Writer
	mu        sync.Mutex
	groups    []string
	attrs     []slog.Attr
	component string
	sessionID string
}

// NewCheckerHandler creates a new CheckerHandler
func NewCheckerHandler(w io.Writer, opts *slog.HandlerOptions, component, sessionID string) *CheckerHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &CheckerHandler{
		opts:      opts,
		output:    w,
		component: component,
		sessionID: sessionID,
	}
}

// Enabled reports whether the handler handles records at the given level
func (h *CheckerHandler) Enabled(ctx context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

// Handle handles the Record
func (h *CheckerHandler) Handle(ctx context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)
	
	// Format timestamp
	if !r.Time.IsZero() {
		buf = append(buf, '[')
		buf = r.Time.AppendFormat(buf, "2006-01-02 15:04:05")
		buf = append(buf, "] "...)
	}
	
	// Format level
	buf = append(buf, r.Level.String()...)
	buf = append(buf, ' ')
	
	// Format component
	if h.component != "" {
		buf = append(buf, '[')
		buf = append(buf, h.component...)
		buf = append(buf, ']')
	}
	
	// Extract and format contextual information from pre-formatted attrs
	var correlationID, taskID, proxyHost string
	var proxyPort, latency, statusCode, retryAttempt int
	var timeout time.Duration
	var errorStr string
	
	// Process pre-formatted attributes from WithAttrs
	for _, a := range h.attrs {
		switch a.Key {
		case "correlation_id":
			correlationID = a.Value.String()
		case "task_id":
			taskID = a.Value.String()
		case "proxy_host":
			proxyHost = a.Value.String()
		case "proxy_port":
			proxyPort = int(a.Value.Int64())
		case "latency_ms":
			latency = int(a.Value.Int64())
		case "status_code":
			statusCode = int(a.Value.Int64())
		case "retry_attempt":
			retryAttempt = int(a.Value.Int64())
		case "timeout_ms":
			timeout = time.Duration(a.Value.Int64()) * time.Millisecond
		case "error":
			errorStr = a.Value.String()
		}
	}
	
	// Process record attributes
	r.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "correlation_id":
			correlationID = a.Value.String()
		case "task_id":
			taskID = a.Value.String()
		case "proxy_host":
			proxyHost = a.Value.String()
		case "proxy_port":
			proxyPort = int(a.Value.Int64())
		case "latency_ms":
			latency = int(a.Value.Int64())
		case "status_code":
			statusCode = int(a.Value.Int64())
		case "retry_attempt":
			retryAttempt = int(a.Value.Int64())
		case "timeout_ms":
			timeout = time.Duration(a.Value.Int64()) * time.Millisecond
		case "error":
			errorStr = a.Value.String()
		}
		return true
	})
	
	// Append contextual information
	if correlationID != "" {
		buf = append(buf, " [CID:"...)
		buf = append(buf, correlationID...)
		buf = append(buf, ']')
	}
	if taskID != "" {
		buf = append(buf, " [TID:"...)
		buf = append(buf, taskID...)
		buf = append(buf, ']')
	}
	if proxyHost != "" {
		buf = append(buf, " [Proxy:"...)
		buf = append(buf, proxyHost...)
		buf = append(buf, ':')
		buf = append(buf, fmt.Sprintf("%d", proxyPort)...)
		buf = append(buf, ']')
	}
	if latency > 0 {
		buf = append(buf, " ["...)
		buf = append(buf, fmt.Sprintf("%dms", latency)...)
		buf = append(buf, ']')
	}
	if statusCode > 0 {
		buf = append(buf, " [HTTP:"...)
		buf = append(buf, fmt.Sprintf("%d", statusCode)...)
		buf = append(buf, ']')
	}
	if retryAttempt > 0 {
		buf = append(buf, " [Retry:"...)
		buf = append(buf, fmt.Sprintf("%d", retryAttempt)...)
		buf = append(buf, ']')
	}
	if timeout > 0 {
		buf = append(buf, " [Timeout:"...)
		buf = append(buf, timeout.String()...)
		buf = append(buf, ']')
	}
	
	// Format message
	buf = append(buf, ' ')
	buf = append(buf, r.Message...)
	
	// Format error if present
	if errorStr != "" {
		buf = append(buf, " - Error: "...)
		buf = append(buf, errorStr...)
	}
	
	// Add other fields
	hasOtherFields := false
	r.Attrs(func(a slog.Attr) bool {
		// Skip already processed fields
		switch a.Key {
		case "correlation_id", "task_id", "proxy_host", "proxy_port",
			"latency_ms", "status_code", "retry_attempt", "timeout_ms", "error":
			return true
		}
		
		if !hasOtherFields {
			buf = append(buf, "\n  Fields: "...)
			hasOtherFields = true
		} else {
			buf = append(buf, ", "...)
		}
		
		buf = append(buf, a.Key...)
		buf = append(buf, '=')
		buf = append(buf, fmt.Sprintf("%v", a.Value.Any())...)
		return true
	})
	
	buf = append(buf, '\n')
	
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.output.Write(buf)
	return err
}

// WithAttrs returns a new Handler with the given attributes added
func (h *CheckerHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)
	
	return &CheckerHandler{
		opts:      h.opts,
		output:    h.output,
		groups:    h.groups,
		attrs:     newAttrs,
		component: h.component,
		sessionID: h.sessionID,
	}
}

// WithGroup returns a new Handler with the given group added
func (h *CheckerHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name
	
	return &CheckerHandler{
		opts:      h.opts,
		output:    h.output,
		groups:    newGroups,
		attrs:     h.attrs,
		component: h.component,
		sessionID: h.sessionID,
	}
}

// StructuredLogger provides structured logging capabilities using slog
type StructuredLogger struct {
	logger     *slog.Logger
	level      LogLevel
	fileOutput *os.File
	jsonFormat bool
	sessionID  string
	component  string
	bufferSize int
	buffer     []LogEntry
	bufferMu   sync.Mutex
}

// LoggerConfig for StructuredLogger
type LoggerConfig struct {
	Level      LogLevel `json:"level"`
	JSONFormat bool     `json:"json_format"`
	OutputFile string   `json:"output_file"`
	BufferSize int      `json:"buffer_size"`
	Component  string   `json:"component"`
}

// NewStructuredLogger creates a new structured logger using slog
func NewStructuredLogger(config LoggerConfig) (*StructuredLogger, error) {
	sessionID := generateSessionID()
	
	var handler slog.Handler
	var fileOutput *os.File
	
	// Set up file output if specified
	if config.OutputFile != "" {
		dir := filepath.Dir(config.OutputFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %v", err)
		}
		
		file, err := os.OpenFile(config.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %v", err)
		}
		fileOutput = file
	}
	
	// Create handler based on format
	opts := &slog.HandlerOptions{
		Level: config.Level.toSlogLevel(),
		AddSource: false,
	}
	
	if config.JSONFormat {
		// Use JSON handler for JSON format
		if fileOutput != nil {
			handler = slog.NewJSONHandler(io.MultiWriter(os.Stdout, fileOutput), opts)
		} else {
			handler = slog.NewJSONHandler(os.Stdout, opts)
		}
	} else {
		// Use custom CheckerHandler for readable text format
		if fileOutput != nil {
			handler = NewCheckerHandler(io.MultiWriter(os.Stdout, fileOutput), opts, config.Component, sessionID)
		} else {
			handler = NewCheckerHandler(os.Stdout, opts, config.Component, sessionID)
		}
	}
	
	logger := slog.New(handler)
	
	return &StructuredLogger{
		logger:     logger,
		level:      config.Level,
		fileOutput: fileOutput,
		jsonFormat: config.JSONFormat,
		sessionID:  sessionID,
		component:  config.Component,
		bufferSize: config.BufferSize,
		buffer:     make([]LogEntry, 0, config.BufferSize),
	}, nil
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().Unix())
}

// Debug logs a debug message
func (sl *StructuredLogger) Debug(message string, fields ...map[string]interface{}) {
	sl.logWithFields(slog.LevelDebug, message, "", fields...)
}

// Info logs an info message
func (sl *StructuredLogger) Info(message string, fields ...map[string]interface{}) {
	sl.logWithFields(slog.LevelInfo, message, "", fields...)
}

// Warn logs a warning message
func (sl *StructuredLogger) Warn(message string, fields ...map[string]interface{}) {
	sl.logWithFields(slog.LevelWarn, message, "", fields...)
}

// Error logs an error message
func (sl *StructuredLogger) Error(message string, err error, fields ...map[string]interface{}) {
	errorStr := ""
	if err != nil {
		errorStr = err.Error()
	}
	sl.logWithFields(slog.LevelError, message, errorStr, fields...)
}

// Fatal logs a fatal message and exits
func (sl *StructuredLogger) Fatal(message string, err error, fields ...map[string]interface{}) {
	errorStr := ""
	if err != nil {
		errorStr = err.Error()
	}
	sl.logWithFields(slog.LevelError+1, message, errorStr, fields...)
	os.Exit(1)
}

// logWithFields is the internal logging method
func (sl *StructuredLogger) logWithFields(level slog.Level, message string, errorStr string, fields ...map[string]interface{}) {
	// Build attributes from fields
	attrs := make([]slog.Attr, 0, 10)
	
	if len(fields) > 0 {
		for _, fieldMap := range fields {
			for k, v := range fieldMap {
				attrs = append(attrs, slog.Any(k, v))
			}
		}
	}
	
	if errorStr != "" {
		attrs = append(attrs, slog.String("error", errorStr))
	}
	
	// Log with context
	sl.logger.LogAttrs(context.Background(), level, message, attrs...)
	
	// Add to buffer if buffering is enabled
	if sl.bufferSize > 0 {
		sl.addToBuffer(level, message, errorStr, fields...)
	}
}

// addToBuffer adds an entry to the internal buffer
func (sl *StructuredLogger) addToBuffer(level slog.Level, message string, errorStr string, fields ...map[string]interface{}) {
	sl.bufferMu.Lock()
	defer sl.bufferMu.Unlock()
	
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   message,
		Component: sl.component,
		Session:   sl.sessionID,
		Error:     errorStr,
	}
	
	if len(fields) > 0 {
		entry.Fields = make(map[string]interface{})
		for _, fieldMap := range fields {
			for k, v := range fieldMap {
				entry.Fields[k] = v
			}
		}
	}
	
	sl.buffer = append(sl.buffer, entry)
	
	// Keep buffer within size limit
	if len(sl.buffer) > sl.bufferSize {
		sl.buffer = sl.buffer[len(sl.buffer)-sl.bufferSize:]
	}
}

// GetRecentLogs returns recent log entries from the buffer
func (sl *StructuredLogger) GetRecentLogs(limit int) []LogEntry {
	sl.bufferMu.Lock()
	defer sl.bufferMu.Unlock()
	
	if limit <= 0 || limit > len(sl.buffer) {
		limit = len(sl.buffer)
	}
	
	start := len(sl.buffer) - limit
	if start < 0 {
		start = 0
	}
	
	result := make([]LogEntry, limit)
	copy(result, sl.buffer[start:])
	return result
}

// SetLevel changes the logging level
func (sl *StructuredLogger) SetLevel(level LogLevel) {
	sl.level = level
}

// SetComponent changes the component name
func (sl *StructuredLogger) SetComponent(component string) {
	sl.component = component
}

// Close closes the logger and any file handles
func (sl *StructuredLogger) Close() error {
	if sl.fileOutput != nil {
		return sl.fileOutput.Close()
	}
	return nil
}

// LogCheckerEvent logs checker-specific events
func (sl *StructuredLogger) LogCheckerEvent(eventType string, result types.CheckResult, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	
	fields["event_type"] = eventType
	fields["combo"] = result.Combo.Username
	fields["config"] = result.Config
	fields["status"] = string(result.Status)
	fields["latency"] = result.Latency
	
	if result.Proxy != nil {
		fields["proxy"] = fmt.Sprintf("%s:%d", result.Proxy.Host, result.Proxy.Port)
	}
	
	sl.Info(fmt.Sprintf("Checker event: %s", eventType), fields)
}

// LogProxyEvent logs proxy-related events
func (sl *StructuredLogger) LogProxyEvent(eventType string, proxy types.Proxy, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	
	fields["event_type"] = eventType
	fields["proxy_host"] = proxy.Host
	fields["proxy_port"] = proxy.Port
	fields["proxy_type"] = string(proxy.Type)
	fields["proxy_score"] = proxy.Score
	fields["proxy_quality"] = string(proxy.Quality)
	
	if proxy.Location != nil {
		fields["proxy_country"] = proxy.Location.Country
	}
	
	sl.Info(fmt.Sprintf("Proxy event: %s", eventType), fields)
}

// ExportLogs exports recent logs to a file
func (sl *StructuredLogger) ExportLogs(filename string, limit int) error {
	logs := sl.GetRecentLogs(limit)
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write as JSON
	fmt.Fprintf(file, "{\n")
	fmt.Fprintf(file, "  \"exported_at\": \"%s\",\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(file, "  \"session_id\": \"%s\",\n", sl.sessionID)
	fmt.Fprintf(file, "  \"total_logs\": %d,\n", len(logs))
	fmt.Fprintf(file, "  \"logs\": [\n")
	
	for i, log := range logs {
		fmt.Fprintf(file, "    {")
		fmt.Fprintf(file, "\"timestamp\":\"%s\",", log.Timestamp.Format(time.RFC3339))
		fmt.Fprintf(file, "\"level\":\"%s\",", log.Level)
		fmt.Fprintf(file, "\"message\":\"%s\"", log.Message)
		if log.Error != "" {
			fmt.Fprintf(file, ",\"error\":\"%s\"", log.Error)
		}
		fmt.Fprintf(file, "}")
		if i < len(logs)-1 {
			fmt.Fprintf(file, ",")
		}
		fmt.Fprintf(file, "\n")
	}
	
	fmt.Fprintf(file, "  ]\n")
	fmt.Fprintf(file, "}\n")
	
	return nil
}

// LogWithCorrelation logs with correlation ID for request tracing
func (sl *StructuredLogger) LogWithCorrelation(level LogLevel, message string, correlationID string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["correlation_id"] = correlationID
	
	switch level {
	case DEBUG:
		sl.Debug(message, fields)
	case INFO:
		sl.Info(message, fields)
	case WARN:
		sl.Warn(message, fields)
	case ERROR:
		sl.Error(message, nil, fields)
	case FATAL:
		sl.Fatal(message, nil, fields)
	}
}

// LogNetworkRequest logs network request details with timeout tracking
func (sl *StructuredLogger) LogNetworkRequest(method, url string, statusCode int, latency time.Duration, proxy *types.Proxy, correlationID string, err error) {
	logger := sl.logger.With(
		slog.String("correlation_id", correlationID),
		slog.Int("status_code", statusCode),
		slog.Int("latency_ms", int(latency.Milliseconds())),
	)
	
	if proxy != nil {
		logger = logger.With(
			slog.String("proxy_host", proxy.Host),
			slog.Int("proxy_port", proxy.Port),
		)
	}
	
	message := fmt.Sprintf("Network request: %s %s", method, url)
	
	if err != nil {
		logger.Error(message, slog.String("error", err.Error()))
	} else {
		logger.Info(message)
	}
	
	// Add to buffer
	if sl.bufferSize > 0 {
		fields := map[string]interface{}{
			"method":         method,
			"url":            url,
			"status_code":    statusCode,
			"latency_ms":     latency.Milliseconds(),
			"correlation_id": correlationID,
		}
		if proxy != nil {
			fields["proxy_host"] = proxy.Host
			fields["proxy_port"] = proxy.Port
		}
		if err != nil {
			sl.addToBuffer(slog.LevelError, message, err.Error(), fields)
		} else {
			sl.addToBuffer(slog.LevelInfo, message, "", fields)
		}
	}
}

// LogProxySelection logs proxy selection decisions
func (sl *StructuredLogger) LogProxySelection(strategy string, proxy *types.Proxy, alternatives int, correlationID string) {
	logger := sl.logger.With(
		slog.String("correlation_id", correlationID),
		slog.String("strategy", strategy),
		slog.Int("alternatives", alternatives),
	)
	
	if proxy != nil {
		logger = logger.With(
			slog.String("proxy_host", proxy.Host),
			slog.Int("proxy_port", proxy.Port),
			slog.Float64("proxy_score", proxy.Score),
		)
	}
	
	logger.Debug(fmt.Sprintf("Proxy selected using %s strategy", strategy))
}

// LogHealthCheck logs health check results
func (sl *StructuredLogger) LogHealthCheck(proxy *types.Proxy, success bool, latency time.Duration, err error) {
	logger := sl.logger.With(
		slog.String("proxy_host", proxy.Host),
		slog.Int("proxy_port", proxy.Port),
		slog.Bool("success", success),
		slog.Int("latency_ms", int(latency.Milliseconds())),
	)
	
	message := fmt.Sprintf("Health check for proxy %s:%d", proxy.Host, proxy.Port)
	
	if !success {
		if err != nil {
			logger.Warn(message, slog.String("error", err.Error()))
		} else {
			logger.Warn(message)
		}
	} else {
		logger.Info(message)
	}
}

// LogTimeout logs timeout events with details
func (sl *StructuredLogger) LogTimeout(operation string, timeout time.Duration, correlationID string, proxy *types.Proxy) {
	logger := sl.logger.With(
		slog.String("correlation_id", correlationID),
		slog.String("operation", operation),
		slog.Int("timeout_ms", int(timeout.Milliseconds())),
	)
	
	if proxy != nil {
		logger = logger.With(
			slog.String("proxy_host", proxy.Host),
			slog.Int("proxy_port", proxy.Port),
		)
	}
	
	logger.Warn(fmt.Sprintf("Operation timeout: %s (%.2fs)", operation, timeout.Seconds()))
}

// LogRetryAttempt logs retry attempts with context
func (sl *StructuredLogger) LogRetryAttempt(operation string, attempt int, maxAttempts int, correlationID string, lastError error) {
	logger := sl.logger.With(
		slog.String("correlation_id", correlationID),
		slog.String("operation", operation),
		slog.Int("retry_attempt", attempt),
		slog.Int("max_attempts", maxAttempts),
	)
	
	message := fmt.Sprintf("Retry attempt %d/%d for %s", attempt, maxAttempts, operation)
	
	if lastError != nil {
		logger.Info(message, slog.String("error", lastError.Error()))
	} else {
		logger.Info(message)
	}
}

// LogTaskStart logs the start of a task with correlation ID
func (sl *StructuredLogger) LogTaskStart(taskID string, taskType string, correlationID string) {
	sl.logger.With(
		slog.String("task_id", taskID),
		slog.String("task_type", taskType),
		slog.String("correlation_id", correlationID),
	).Info(fmt.Sprintf("Task started: %s", taskType))
}

// LogTaskComplete logs task completion with performance metrics
func (sl *StructuredLogger) LogTaskComplete(taskID string, taskType string, correlationID string, duration time.Duration, success bool, err error) {
	logger := sl.logger.With(
		slog.String("task_id", taskID),
		slog.String("task_type", taskType),
		slog.String("correlation_id", correlationID),
		slog.Int("latency_ms", int(duration.Milliseconds())),
		slog.Bool("success", success),
	)
	
	message := fmt.Sprintf("Task completed: %s (%.2fs)", taskType, duration.Seconds())
	
	if !success && err != nil {
		logger.Error(message, slog.String("error", err.Error()))
	} else {
		logger.Info(message)
	}
}
