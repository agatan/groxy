package groxy

import (
	"fmt"
	"io"
	"net/http"
)

type ProxyServer struct{}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
