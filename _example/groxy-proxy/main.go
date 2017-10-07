package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/agatan/groxy"
)

type logger struct{}

func (logger) Print(args ...interface{}) {
	log.Println(args...)
}

func main() {
	proxy := groxy.New()
	proxy.Logger = logger{}
	if err := http.ListenAndServe(":8888", proxy); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
