package main

import (
	"antifraud/internal/transport"
	"flag"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	health := flag.Bool("healthcheck", false, "check gateway")
	flag.Parse()
	if *health {
		c := http.Client{Timeout: time.Second}
		r, e := c.Get("http://127.0.0.1:8080/v1/state/demo-clean")
		if e != nil {
			os.Exit(1)
		}
		if err := r.Body.Close(); err != nil {
			os.Exit(1)
		}
		if r.StatusCode != 200 {
			os.Exit(1)
		}
		return
	}
	url := os.Getenv("RISK_URL")
	if url == "" {
		url = "http://127.0.0.1:8081"
	}
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	s := &http.Server{Addr: addr, Handler: transport.GatewayRouter(url), ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second}
	log.Fatal(s.ListenAndServe())
}
