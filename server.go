package groxy

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/pkg/errors"
)

// HTTPSAction defines how to act for requests via HTTPS.
type HTTPSAction int

const (
	// HTTPSActionProxy just performs as proxy server. In this behaviour, middlewares are ignored.
	HTTPSActionProxy HTTPSAction = iota
	// HTTPSActionReject rejects all HTTPS requests (returns http.StatusBadRequest).
	HTTPSActionReject
	// HTTPSActionMITM strips SSL encryption.
	// Builtin certificates is not verified, so clients must accept insecure certificates.
	HTTPSActionMITM
)

// ProxyServer is a programmable proxy server instance, behaves as an http.Handler.
type ProxyServer struct {
	// Logger is a logger that prints proxy requests.
	Logger Logger
	// NonProxyRequestHandler handles non-proxy requests.
	// If it's nil, non-proxy requests causes http.StatusBadRequest.
	NonProxyRequestHandler http.Handler
	HTTPSAction            HTTPSAction
	middlewares            []Middleware
}

// Use adds given middlewares to p's middlewares.
func (p *ProxyServer) Use(ms ...Middleware) {
	p.middlewares = append(p.middlewares, ms...)
}

func (p *ProxyServer) log(args ...interface{}) {
	if p.Logger != nil {
		p.Logger.Print(args...)
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
		p.log("failed to pipe connections: ", err)
	}
	dst.CloseWrite()
	src.CloseRead()
}

func (p *ProxyServer) apply(base Handler) Handler {
	for _, m := range p.middlewares {
		base = m(base)
	}
	return base
}

func (p *ProxyServer) proxyHTTPS(w http.ResponseWriter, r *http.Request) {
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

	p.log("accept CONNECT to ", r.URL.Host)
}

func (p *ProxyServer) mitmHTTPS(w http.ResponseWriter, r *http.Request) {
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

	cliConn.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))
	tlsConfig := &tls.Config{InsecureSkipVerify: true, Certificates: []tls.Certificate{groxyCa}}
	rawCli := tls.Server(cliConn, tlsConfig)
	defer rawCli.Close()
	cliReader := bufio.NewReader(rawCli)
	mitmTr := &http.Transport{TLSClientConfig: tlsConfig, Proxy: http.ProxyFromEnvironment}
	handler := p.apply(DefaultHTTPSHandler(mitmTr))
	for {
		req, err := http.ReadRequest(cliReader)
		if err != nil {
			if err == io.EOF {
				break
			}
			p.log("failed to read TLS request: ", err)
			break
		}
		req.URL.Host = req.Host
		req.URL.Scheme = "https"
		resp, err := handler(req)
		if err != nil {
			p.log("failed to read TLS response: ", err)
			break
		}
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			p.log("failed to read respnse body: ", err)
			break
		}
		if _, err := io.WriteString(rawCli, "HTTP/1.1 "+resp.Status+"\r\n"); err != nil {
			p.log("failed to write TLS response: ", err)
			break
		}
		resp.Header.Write(rawCli)
		rawCli.Write([]byte("\r\n"))
		rawCli.Write(body)
	}
}

func (p *ProxyServer) connectHandler(w http.ResponseWriter, r *http.Request) {
	switch p.HTTPSAction {
	case HTTPSActionProxy:
		p.proxyHTTPS(w, r)
	case HTTPSActionReject:
		http.Error(w, "HTTPS request is not allowed", http.StatusBadRequest)
	case HTTPSActionMITM:
		p.mitmHTTPS(w, r)
	default:
		http.Error(w, fmt.Sprintf("unknown HTTPS action: %v", p.HTTPSAction), http.StatusInternalServerError)
	}
}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.log("received request: ", r)
	if r.Method == "CONNECT" {
		p.connectHandler(w, r)
		return
	}
	if !r.URL.IsAbs() {
		if p.NonProxyRequestHandler == nil {
			http.Error(w, "cannot handle non-proxy requests", http.StatusBadRequest)
		} else {
			p.NonProxyRequestHandler.ServeHTTP(w, r)
		}
		return
	}
	proxyr, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("broken request format: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := p.apply(DefaultHTTPHandler)(proxyr)
	if err != nil {
		http.Error(w, fmt.Sprintf("request failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if err := copyResponse(w, resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
