package groxy

// Logger is an interface to log events.
type Logger interface {
	Print(...interface{})
}

type nullLogger struct{}

func (nullLogger) Print(...interface{}) {}

// FuncLogger is a Logger that wraps print function
type FuncLogger func(...interface{})

// Print invokes f with args
func (f FuncLogger) Print(args ...interface{}) {
	f(args...)
}
