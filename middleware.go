package groxy

import "net/http"

type Handler func(*http.Request) (*http.Response, error)
type Middleware func(Handler) Handler

var httpclient = &http.Client{
	CheckRedirect: func(r *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func DefaultHTTPHandler(req *http.Request) (*http.Response, error) {
	return httpclient.Do(req)
}

func DefaultHTTPSHandler(tr *http.Transport) Handler {
	return tr.RoundTrip
}
