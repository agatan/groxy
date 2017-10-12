package groxy

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
)

func ExampleProxyServer() {
	// creating a new proxy server instance.
	p := &ProxyServer{
		// set HTTPS action (default: HTTPSActionProxy)
		HTTPSAction: HTTPSActionProxy,
	}
	// if you want to hijack https connection, you can use:
	// p.HTTPSAction = HTTPSActionMITM

	// ProxyServer implements http.Handler
	proxyserver := httptest.NewServer(p)
	defer proxyserver.Close()

	// Output:
}

func Example() {
	// creating a new proxy server instance.
	p := &ProxyServer{
		// set HTTPS action (default: HTTPSActionProxy)
		HTTPSAction: HTTPSActionProxy,
	}
	// define a middleware that recreates request handler based on the original handler (original handler performs as a proxy).
	pathLogger := func(h Handler) Handler {
		return func(r *http.Request) (*http.Response, error) {
			fmt.Println(r.URL.Path)
			resp, err := h(r)
			if err != nil {
				return nil, err
			}
			fmt.Println(resp.StatusCode)
			return resp, nil
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
	// 200
	// Hello!
}

func ExampleProxyServer_Use_mitm() {
	var p ProxyServer
	p.HTTPSAction = HTTPSActionMITM
	// hijack request!
	p.Use(func(h Handler) Handler {
		return func(req *http.Request) (*http.Response, error) {
			message := "hijack!"
			body, _ := ioutil.ReadAll(req.Body)
			fmt.Printf("original request: %v\n", string(body))
			_ = req.Body.Close()
			req.Body = ioutil.NopCloser(strings.NewReader(message))
			req.ContentLength = int64(len(message))
			return h(req)
		}
	})

	proxyserver := httptest.NewServer(&p)
	defer proxyserver.Close()
	proxyurl, err := url.Parse(proxyserver.URL)
	if err != nil {
		panic(err)
	}

	// setup a dummy echo server.
	testserver := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		w.Write(body)
	}))
	defer testserver.Close()

	// request via the proxy.
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(proxyurl),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	// Post `message!` to the server.
	resp, err := client.Post(testserver.URL+"/post", "application/json", strings.NewReader("message!"))
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	// if there are no proxy server, body is `message!`.
	fmt.Printf("response: %v\n", string(body))

	// Output:
	// original request: message!
	// response: hijack!
}
