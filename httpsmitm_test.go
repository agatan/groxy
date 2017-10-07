package groxy

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHTTPSManInTheMiddle(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
