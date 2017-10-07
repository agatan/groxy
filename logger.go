package groxy

type Logger interface {
	Print(...interface{})
}

type nullLogger struct{}

func (nullLogger) Print(...interface{}) {}
