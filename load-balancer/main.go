package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// startBackendServer simulates an internal API  or wen app
// It listens on the port specified and is not meant to be accessed directly by the outside world
func startBackendServer(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from the hidden backend server on %s! You requested path: %s\n", port, r.URL.Path)
	})
	log.Printf("backend server is running internally no %s\n", port)
	// we use log.fatal so that program crashes early if the port is already in use
	log.Fatal(http.ListenAndServe(port, mux))
}

func main() {
	// 1. Start a single hidden backend in a backgrounf goroutine
	go startBackendServer(":9091")
	//2. Parse the destination URL (where the proxxy should send traffic)
	// We check for errors immediately (Crash Early Principle)

	targetURL, err := url.Parse("http://localhost:9091")
	if err != nil {
		log.Fatal("Error prasing backend URL:", err)
	}

	//3. Initialize the reverse proxy using Go's built-in library
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 4. Start the proxy server on port 8080
	// ANy traffic hitting 8080 will be swallowed by 'proxy' nd spat out to 'targetURL'

	log.Println("LoadBalancer (Proxy) listening on :8080....")
	log.Fatal(http.ListenAndServe(":8080", proxy))
}
