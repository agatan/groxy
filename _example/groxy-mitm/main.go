package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/agatan/groxy"
)

func logging(h groxy.Handler) groxy.Handler {
	return func(req *http.Request) (*http.Response, error) {
		resp, err := h(req)
		if err != nil {
			return nil, err
		}
		fmt.Printf("%#v\n", resp)
		return resp, nil
	}
}

func main() {
	var proxy groxy.ProxyServer
	proxy.HTTPSAction = groxy.HTTPSActionMITM
	proxy.Use(logging)

	if err := http.ListenAndServe(":8888", &proxy); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
