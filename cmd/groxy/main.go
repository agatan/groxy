package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/agatan/groxy"
)

func main() {
	proxy := new(groxy.ProxyServer)
	if err := http.ListenAndServe(":8888", proxy); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
