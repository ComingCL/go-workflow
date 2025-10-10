package workflow

// Logger defines the interface for logging operations
type Logger interface {
	// Info logs an info level message with optional key-value pairs
	Info(msg string, keysAndValues ...interface{})

	// Error logs an error level message with optional key-value pairs
	Error(err error, msg string, keysAndValues ...interface{})
}
