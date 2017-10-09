package groxy

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
)

func Example_SimpleProxy() {
	// creating a new proxy server instance.
	p := New()
	// set HTTPS action (default: HTTPSActionProxy)
	p.HTTPSAction = HTTPSActionProxy
	// ProxyServer implements http.Handler
	proxyserver := httptest.NewServer(p)
	defer proxyserver.Close()

	// Output:
}

func Example_Middleware() {
	p := New()
	// define a middleware that recreates request handler based on the original handler (original handler performs just a proxy).
	pathLogger := func(h Handler) Handler {
		return func(r *http.Request) (*http.Response, error) {
			fmt.Println(r.URL.Path)
			return h(r)
		}
	}
	// set the middleware.
	p.Use(pathLogger)

	proxyserver := httptest.NewServer(p)
	defer proxyserver.Close()
	proxyurl, _ := url.Parse(proxyserver.URL)

	// setup a dummy server.
	testserver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello!"))
	}))
	defer testserver.Close()

	// request via the proxy.
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyurl)}}
	resp, _ := client.Get(testserver.URL + "/foo/bar")
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	fmt.Println(string(body))

	// Output:
	// /foo/bar
	// Hello!
}

func Example_ManInTheMiddle() {
	p := New()
	// set HTTPSAction to HTTPSActionMITM, that enables man in the middle hijacking.
	p.HTTPSAction = HTTPSActionMITM
	p.Use(func(h Handler) Handler {
		return func(r *http.Request) (*http.Response, error) {
			message := "hijack!"
			r.Body.Close()
			r.Body = ioutil.NopCloser(strings.NewReader(message))
			r.ContentLength = int64(len(message))
			return h(r)
		}
	})

	proxyserver := httptest.NewServer(p)
	defer proxyserver.Close()
	proxyurl, _ := url.Parse(proxyserver.URL)

	// setup a dummy echo server.
	testserver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		r.Body.Close()
		w.Write(body)
	}))
	defer testserver.Close()

	// request via the proxy.
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyurl)}}
	// Post `message!` to the server.
	resp, _ := client.Post(testserver.URL, "", strings.NewReader("message!"))
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	// if there are no proxy server, body is `message!`.
	fmt.Println(string(body))

	// Output:
	// hijack!
}