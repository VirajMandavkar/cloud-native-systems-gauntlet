package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

// Define the Load Balacer strucct - Holding the routing state
type LoadBalancer struct {
	port            string
	servers         BackendHeap
	roundRobinCount int
	mu              sync.Mutex
}

// Defining the backend struct
type Backend struct {
	URL         *url.URL
	Alive       bool
	ActiveConns int
	Index       int
	mu          sync.RWMutex
}

// Defining Min.Heap
type BackendHeap []*Backend

// Len() telling the heap how may servers exist
func (h BackendHeap) Len() int {
	return len(h)
}

// startBackendServer simulates an internal API  or wen app
// It listens on the port specified and is not meant to be accessed directly by the outside world
func startBackendServer(port string) {
	mux := http.NewServeMux()
	isHealthy := true // Server will start as healthy

	mux.HandleFunc("/toggle", func(w http.ResponseWriter, r *http.Request) {
		isHealthy = !isHealthy
		fmt.Fprintf(w, "Backend %s health state changed to: %v\n", port, isHealthy)
	})

	// The main handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !isHealthy {
			http.Error(w, "Simulated Crash!", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Hello from the hidden backend server on %s ! You requested path: %s\n", port, r.URL.Path)
	})

	log.Printf("Backend server is running internally on %s\n", port)
	log.Fatal(http.ListenAndServe(port, mux))
}

// Pinger

func (lb *LoadBalancer) healthCheck() {
	ticker := time.NewTicker(3 * time.Second)

	for range ticker.C {
		log.Println("Runnig health check on backends...") // will be remove later - its just kept for testing purpose

		for _, b := range lb.servers {
			client := http.Client{
				Timeout: 2 * time.Second,
			}
			resp, err := client.Get(b.URL.String())
			b.mu.Lock()

			if err != nil || resp.StatusCode != http.StatusOK {
				if b.Alive {
					log.Printf("Backend %s is DOWN!! Marking unhealthy.\n", b.URL)
					b.Alive = false
				}

			} else {
				if !b.Alive {
					log.Printf("Backend %s is back UP ! Marking it as healthy.\n", b.URL)
					b.Alive = true
				}
				resp.Body.Close()
			}
			b.mu.Unlock()
		}
	}
}

// getNextPeer is a helper function that ONLY picks the server.
// It locks and unlocks as fast as possible.
func (lb *LoadBalancer) getNextPeer() *Backend {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i := 0; i < len(lb.servers); i++ {
		serverIndex := lb.roundRobinCount % len(lb.servers)
		lb.roundRobinCount++

		backend := lb.servers[serverIndex]

		// Use RLock for safe concurrent reading of the boolean
		backend.mu.RLock()
		isAlive := backend.Alive
		backend.mu.RUnlock()

		if isAlive {
			return backend
		}
	}

	// If we checked all servers and none are alive
	return nil
}

//Design with contracts : implement the http.handler interface. beacuse this struct has a serveHTTP method, Go knows it can act as a web server.

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Get the next healthy backend (this handles its own locking)
	target := lb.getNextPeer()

	if target == nil {
		http.Error(w, "no healthy backends available", http.StatusServiceUnavailable)
		return
	}

	// 2. Forward the request!
	// Notice that NO locks are held here. 10,000 goroutines can run this line simultaneously.
	fmt.Printf("Routing request to %s\n", target.URL)
	proxy := httputil.NewSingleHostReverseProxy(target.URL)
	proxy.ServeHTTP(w, r)
}

func main() {
	// Starting 3 backends in the background

	backends := []string{":9091", ":9092", ":9093"}
	for _, port := range backends {
		go startBackendServer(port)
	}

	// parse the strings into actual url.URL ogjects
	var servers []*Backend
	for _, port := range backends {
		u, _ := url.Parse("http://localhost" + port)
		backend := &Backend{
			URL:   u,
			Alive: true,
		}
		servers = append(servers, backend)
	}
	//initialization of load balancer
	lb := &LoadBalancer{
		port:    ":8080",
		servers: servers,
	}

	//Start the background health check worker
	go lb.healthCheck()

	log.Printf("LoadBalancer listening on %s.. \n", lb.port)

	log.Fatal(http.ListenAndServe(lb.port, lb))
}
