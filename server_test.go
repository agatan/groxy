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

	proxy := New()
	proxyserver := httptest.NewServer(proxy)
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

	proxy := New()
	proxyserver := httptest.NewServer(proxy)
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

	proxy := New()
	proxy.HTTPSAction = HTTPSActionMITM
	proxyserver := httptest.NewServer(proxy)
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

func TestHTTPSRefuse(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		w.Write(body)
	}))
	defer ts.Close()

	proxy := New()
	proxy.HTTPSAction = HTTPSActionRefuse
	proxyserver := httptest.NewServer(proxy)
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
