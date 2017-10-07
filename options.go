package groxy

type Option func(*ProxyServer) Option

type Logger interface {
	Print(...interface{})
}

type nullLogger struct{}

func (nullLogger) Print(...interface{}) {}

func Log(l Logger) Option {
	return func(p *ProxyServer) Option {
		prev := p.logger
		p.logger = l
		return Log(prev)
	}
}
