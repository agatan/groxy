package groxy

type Logger interface {
	Print(...interface{})
}

type nullLogger struct{}

func (nullLogger) Print(...interface{}) {}

type FuncLogger func(...interface{})

func (f FuncLogger) Print(args ...interface{}) {
	f(args...)
}
