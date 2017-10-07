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
