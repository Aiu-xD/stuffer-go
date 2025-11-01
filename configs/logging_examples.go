package configs

import "universal-checker/internal/logger"

// DevelopmentLoggerConfig returns a logger configuration optimized for development
// Features: DEBUG level, readable text format, file output, buffering enabled
func DevelopmentLoggerConfig(component string) logger.LoggerConfig {
	return logger.LoggerConfig{
		Level:      logger.DEBUG,
		JSONFormat: false,
		OutputFile: "logs/dev.log",
		BufferSize: 1000,
		Component:  component,
	}
}

// ProductionLoggerConfig returns a logger configuration optimized for production
// Features: INFO level, JSON format, file output, large buffer
func ProductionLoggerConfig(component string) logger.LoggerConfig {
	return logger.LoggerConfig{
		Level:      logger.INFO,
		JSONFormat: true,
		OutputFile: "logs/production.log",
		BufferSize: 5000,
		Component:  component,
	}
}

// TestLoggerConfig returns a logger configuration optimized for testing
// Features: DEBUG level, text format, stdout only, no buffering
func TestLoggerConfig(component string) logger.LoggerConfig {
	return logger.LoggerConfig{
		Level:      logger.DEBUG,
		JSONFormat: false,
		OutputFile: "",
		BufferSize: 0,
		Component:  component,
	}
}

// DebugLoggerConfig returns a logger configuration for intensive debugging
// Features: DEBUG level, text format, file output, large buffer for analysis
func DebugLoggerConfig(component string) logger.LoggerConfig {
	return logger.LoggerConfig{
		Level:      logger.DEBUG,
		JSONFormat: false,
		OutputFile: "logs/debug.log",
		BufferSize: 10000,
		Component:  component,
	}
}

// QuietLoggerConfig returns a logger configuration for minimal logging
// Features: ERROR level only, JSON format, file output, small buffer
func QuietLoggerConfig(component string) logger.LoggerConfig {
	return logger.LoggerConfig{
		Level:      logger.ERROR,
		JSONFormat: true,
		OutputFile: "logs/errors.log",
		BufferSize: 500,
		Component:  component,
	}
}

// ConsoleOnlyLoggerConfig returns a logger configuration for console output only
// Features: INFO level, text format, no file output, no buffering
func ConsoleOnlyLoggerConfig(component string) logger.LoggerConfig {
	return logger.LoggerConfig{
		Level:      logger.INFO,
		JSONFormat: false,
		OutputFile: "",
		BufferSize: 0,
		Component:  component,
	}
}
