package groxy

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/pkg/errors"
)

type ProxyServer struct {
	Logger  *log.Logger
	Handler http.Handler
	client  *http.Client
}

func New() *ProxyServer {
	return &ProxyServer{
		client: &http.Client{
			CheckRedirect: func(r *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (p *ProxyServer) logf(f string, args ...interface{}) {
	if p.Logger == nil {
		log.Printf(f, args...)
	} else {
		p.Logger.Printf(f, args...)
	}
}

func copyResponse(dst http.ResponseWriter, src *http.Response) error {
	dstHeader := dst.Header()
	for k := range dstHeader {
		dstHeader.Del(k)
	}
	for k, vs := range src.Header {
		for _, v := range vs {
			dstHeader.Add(k, v)
		}
	}
	dst.WriteHeader(src.StatusCode)
	if _, err := io.Copy(dst, src.Body); err != nil {
		return errors.Wrap(err, "failed to copy response body")
	}
	return nil
}

func (p *ProxyServer) pipeConn(dst, src *net.TCPConn) {
	if _, err := io.Copy(dst, src); err != nil {
		p.logf("failed to pipe connections: %v", err)
	}
	dst.CloseWrite()
	src.CloseRead()
}

func (p *ProxyServer) handleHTTPS(w http.ResponseWriter, r *http.Request) {
	hij, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "cannot hijack https request", http.StatusInternalServerError)
		return
	}
	cliConn, _, err := hij.Hijack()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to hijack https connection: %v", err), http.StatusInternalServerError)
		return
	}

	cliConn.Write([]byte("HTTP/1.0 200 OK \r\n\r\n"))

	dstConn, err := net.Dial("tcp", r.URL.Host)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to connect the destination server: %v", err), http.StatusBadGateway)
		return
	}
	dstTCPConn := dstConn.(*net.TCPConn)
	cliTCPConn := cliConn.(*net.TCPConn)

	go p.pipeConn(dstTCPConn, cliTCPConn)
	go p.pipeConn(cliTCPConn, dstTCPConn)

	p.logf("accept CONNECT to %s", r.URL.Host)
}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.logf("received request: %#v", r)
	if r.Method == "CONNECT" {
		p.handleHTTPS(w, r)
		return
	}
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

	resp, err := p.client.Do(proxyr)
	if err != nil {
		http.Error(w, fmt.Sprintf("ruest failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if err := copyResponse(w, resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
