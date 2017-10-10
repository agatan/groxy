package groxy

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHTTPProxyWithMiddleware(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		w.Write(body)
	}))
	defer ts.Close()

	var proxy ProxyServer
	rewriteMessage := "rewrite"
	proxy.Use(func(h Handler) Handler {
		return func(req *http.Request) (*http.Response, error) {
			_ = req.Body.Close()
			req.Body = ioutil.NopCloser(strings.NewReader(rewriteMessage))
			req.ContentLength = int64(len(rewriteMessage))
			return h(req)
		}
	})
	proxyserver := httptest.NewServer(&proxy)
	defer proxyserver.Close()
	proxyurl, err := url.Parse(proxyserver.URL)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyurl)}}
	resp, err := client.Post(ts.URL+"/post", "application/json", strings.NewReader("original message"))
	if err != nil {
		t.Fatalf("failed to request via proxy: %v", err)
	}
	gotbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	if string(gotbody) != rewriteMessage {
		t.Errorf("expected response body is %q, but got %q", rewriteMessage, string(gotbody))
	}
}

func TestHTTPSMitmWithMiddleware(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		w.Write(body)
	}))
	defer ts.Close()

	var proxy ProxyServer
	rewriteMessage := "rewrite"
	proxy.HTTPSAction = HTTPSActionMITM
	proxy.Use(func(h Handler) Handler {
		return func(req *http.Request) (*http.Response, error) {
			_ = req.Body.Close()
			req.Body = ioutil.NopCloser(strings.NewReader(rewriteMessage))
			req.ContentLength = int64(len(rewriteMessage))
			return h(req)
		}
	})
	proxyserver := httptest.NewServer(&proxy)
	defer proxyserver.Close()
	proxyurl, err := url.Parse(proxyserver.URL)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(proxyurl),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Post(ts.URL+"/post", "application/json", strings.NewReader("original message"))
	if err != nil {
		t.Fatalf("failed to request via proxy: %v", err)
	}
	gotbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	if string(gotbody) != rewriteMessage {
		t.Errorf("expected response body is %q, but got %q", rewriteMessage, string(gotbody))
	}
}
