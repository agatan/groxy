package groxy

import "net/http"

// Handler handles http.Request and somehow generate http.Response or error.
type Handler func(*http.Request) (*http.Response, error)

// Middleware wraps original Handler and create new Handler.
type Middleware func(Handler) Handler

var httpclient = &http.Client{
	CheckRedirect: func(r *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// DefaultHTTPHandler pass the request to the target server, and returns its response or error.
func DefaultHTTPHandler(req *http.Request) (*http.Response, error) {
	return httpclient.Do(req)
}

// DefaultHTTPSHandler pass the request to the target server, and returns its response or error.
func DefaultHTTPSHandler(tr *http.Transport) Handler {
	return tr.RoundTrip
}
