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

func TestHTTPProxy(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		w.Write(body)
	}))
	defer ts.Close()

	var proxy ProxyServer
	proxyserver := httptest.NewServer(&proxy)
	defer proxyserver.Close()
	proxyurl, err := url.Parse(proxyserver.URL)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyurl)}}
	body := `{"message": "Hello, world!"}`
	resp, err := client.Post(ts.URL+"/post", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("failed to request via proxy: %v", err)
	}
	gotbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	if string(gotbody) != body {
		t.Errorf("expected response body is %q, but got %q", body, string(gotbody))
	}
}

func TestHTTPSProxy(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		w.Write(body)
	}))
	defer ts.Close()

	var proxy ProxyServer
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
	body := `{"message": "Hello, world!"}`
	resp, err := client.Post(ts.URL+"/post", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("failed to request via proxy: %v", err)
	}
	gotbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	if string(gotbody) != body {
		t.Errorf("expected response body is %q, but got %q", body, string(gotbody))
	}
}

func TestHTTPSManInTheMiddle(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		w.Write(body)
	}))
	defer ts.Close()

	var proxy ProxyServer
	proxy.HTTPSAction = HTTPSActionMITM
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
	body := `{"message": "Hello, world!"}`
	resp, err := client.Post(ts.URL+"/post", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("failed to request via proxy: %v", err)
	}
	gotbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	if string(gotbody) != body {
		t.Errorf("expected response body is %q, but got %q", body, string(gotbody))
	}
}

func TestHTTPSReject(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		w.Write(body)
	}))
	defer ts.Close()

	var proxy ProxyServer
	proxy.HTTPSAction = HTTPSActionReject
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
	body := `{"message": "Hello, world!"}`
	resp, err := client.Post(ts.URL+"/post", "application/json", strings.NewReader(body))
	if err == nil && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("https request should be refused, but got response %v and error %v", resp, err)
	}
}

func TestRejectNonProxyRequest(t *testing.T) {
	var proxy ProxyServer
	proxyserver := httptest.NewServer(&proxy)
	defer proxyserver.Close()

	resp, err := http.Get(proxyserver.URL)
	if err != nil {
		t.Fatalf("failed to request via proxy: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status code is %v, but got %v", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestNonProxyRequestWithHandler(t *testing.T) {
	var proxy ProxyServer
	reply := "Reply for non proxy requests"
	proxy.NonProxyRequestHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(reply))
	})
	proxyserver := httptest.NewServer(&proxy)
	defer proxyserver.Close()

	resp, err := http.Get(proxyserver.URL)
	if err != nil {
		t.Fatalf("failed to request via proxy: %v", err)
	}
	gotbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	if string(gotbody) != reply {
		t.Errorf("expected response body is %q, but got %q", reply, string(gotbody))
	}
}
