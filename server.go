package groxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

type ProxyServer struct {
	Logger  *log.Logger
	Handler http.Handler
}

func (p *ProxyServer) logf(f string, args ...interface{}) {
	if p.Logger == nil {
		log.Printf(f, args...)
	} else {
		p.Logger.Panicf(f, args...)
	}
}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.logf("received request: %#v", r)
	if !r.URL.IsAbs() {
		if p.Handler == nil {
			http.Error(w, "cannot handle non-proxy requests", http.StatusBadRequest)
		} else {
			p.Handler.ServeHTTP(w, r)
		}
		return
	}
	proxyr, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("broken request format: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := http.DefaultClient.Do(proxyr)
	if err != nil {
		http.Error(w, fmt.Sprintf("ruest failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if _, err := io.Copy(w, resp.Body); err != nil {
		http.Error(w, fmt.Sprintf("failed to copy response: %v", err), http.StatusInternalServerError)
		return
	}
}
