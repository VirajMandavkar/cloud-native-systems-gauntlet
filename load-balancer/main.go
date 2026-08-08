package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
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

// Define the Load Balacer strucct - Holding the routing state
type LoadBalancer struct {
	port            string
	servers         []*url.URL
	roundRobinCount int
	mu              sync.Mutex
}

//Design with contracts : implement the http.handler interface. beacuse this struct has a serveHTTP method, Go knows it can act as a web server.

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//Lock the mutex to safely pick the next server
	lb.mu.Lock()

	//Modulo arithmetic ensure the index loops back to 0 when it hits the end of the slice
	serverIndex := lb.roundRobinCount % len(lb.servers)
	targetURL := lb.servers[serverIndex]

	lb.roundRobinCount++
	lb.mu.Unlock()
	fmt.Printf("Routing request to %s\n", targetURL)

	// Create a reverse proxy dynamically and forward the request
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ServeHTTP(w, r)
}

func main() {
	// Starting 3 backends in the background

	backends := []string{":9091", ":9092", ":9093"}
	for _, port := range backends {
		go startBackendServer(port)
	}

	// parse the strings into actual url.URL ogjects
	var servers []*url.URL
	for _, port := range backends {
		u, _ := url.Parse("http://localhost" + port)
		servers = append(servers, u)
	}
	//initialization of load balancer
	lb := &LoadBalancer{
		port:    ":8080",
		servers: servers,
	}

	log.Printf("LoadBalancer listening on %s.. \n", lb.port)

	log.Fatal(http.ListenAndServe(lb.port, lb))
}
