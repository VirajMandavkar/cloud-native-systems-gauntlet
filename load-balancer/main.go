package main

import (
	"container/heap"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

// Define the Load Balancer struct - Holding the routing state
type LoadBalancer struct {
	port    string
	servers BackendHeap
	mu      sync.Mutex // Protects the heap operations
}

// Defining the backend struct
type Backend struct {
	URL         *url.URL
	Alive       bool
	ActiveConns int
	Index       int
	mu          sync.RWMutex // Protects individual backend state (like during ping)
}

// Defining Min-Heap
type BackendHeap []*Backend

// 1. Len() telling the heap how many servers exist
func (h BackendHeap) Len() int {
	return len(h)
}

// 2. Less() tells the heap how to sort the servers
func (h BackendHeap) Less(i, j int) bool {
	// Gravity trap: Dead servers are infinitely heavy
	if !h[i].Alive {
		return false
	}
	if !h[j].Alive {
		return true
	}
	return h[i].ActiveConns < h[j].ActiveConns
}

// 3. Swap() trades places and updates the Index
func (h BackendHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}

// 4. Push() adds a new backend
func (h *BackendHeap) Push(x any) {
	n := len(*h)
	backend := x.(*Backend)
	backend.Index = n
	*h = append(*h, backend)
}

// 5. Pop() removes the last backend
func (h *BackendHeap) Pop() any {
	old := *h
	n := len(old)
	backend := old[n-1]
	old[n-1] = nil // Avoid memory leak
	backend.Index = -1
	*h = old[0 : n-1]
	return backend
}

// startBackendServer simulates an internal API or web app
func startBackendServer(port string) {
	mux := http.NewServeMux()
	isHealthy := true // Server will start as healthy

	mux.HandleFunc("/toggle", func(w http.ResponseWriter, r *http.Request) {
		isHealthy = !isHealthy
		fmt.Fprintf(w, "Backend %s health state changed to: %v\n", port, isHealthy)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !isHealthy {
			http.Error(w, "Simulated Crash!", http.StatusInternalServerError)
			return
		}
		// Simulate some work so we can actually see connections pile up
		time.Sleep(100 * time.Millisecond)
		fmt.Fprintf(w, "Hello from the hidden backend server on %s ! You requested path: %s\n", port, r.URL.Path)
	})

	log.Printf("Backend server is running internally on %s\n", port)
	log.Fatal(http.ListenAndServe(port, mux))
}

// healthCheck runs continuously in the background
func (lb *LoadBalancer) healthCheck() {
	ticker := time.NewTicker(3 * time.Second)

	for range ticker.C {
		// Pinging each server
		for _, b := range lb.servers {
			client := http.Client{
				Timeout: 2 * time.Second,
			}
			resp, err := client.Get(b.URL.String())

			// We need to lock the main load balancer mutex because if we change b.Alive,
			// we MUST call heap.Fix(), which modifies the shared heap structure.
			lb.mu.Lock()

			if err != nil || resp.StatusCode != http.StatusOK {
				if b.Alive {
					log.Printf("Backend %s is DOWN!! Marking unhealthy.\n", b.URL.String())
					b.Alive = false
					heap.Fix(&lb.servers, b.Index) // Sink it to the bottom
				}
			} else {
				if !b.Alive {
					log.Printf("Backend %s is back UP ! Marking it as healthy.\n", b.URL.String())
					b.Alive = true
					heap.Fix(&lb.servers, b.Index) // Float it back to the top
				}
				resp.Body.Close()
			}

			lb.mu.Unlock()
		}
	}
}

// ServeHTTP implements the reverse proxy routing
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Get the best healthy backend and increment its connections
	lb.mu.Lock()
	target := lb.servers[0]

	if !target.Alive {
		lb.mu.Unlock()
		http.Error(w, "no healthy backend available", http.StatusServiceUnavailable)
		return
	}

	target.ActiveConns++
	heap.Fix(&lb.servers, target.Index)
	lb.mu.Unlock() // Unlock immediately so other requests aren't blocked!

	fmt.Printf("Routing to %s (Active Connections: %d)\n", target.URL.String(), target.ActiveConns)

	// 2. Forward the traffic (Blocking network call)
	proxy := httputil.NewSingleHostReverseProxy(target.URL)
	proxy.ServeHTTP(w, r)

	// 3. Cleanup: Decrement connections after the request finishes
	lb.mu.Lock()
	target.ActiveConns--
	heap.Fix(&lb.servers, target.Index)
	lb.mu.Unlock()
}

func main() {
	backends := []string{":9091", ":9092", ":9093"}
	for _, port := range backends {
		go startBackendServer(port)
	}

	var servers BackendHeap
	for i, port := range backends {
		u, _ := url.Parse("http://localhost" + port)
		backend := &Backend{
			URL:   u,
			Alive: true,
			Index: i, // CRITICAL: You must set the initial index before pushing to the heap
		}
		servers = append(servers, backend)
	}

	// Initialize the heap logic
	heap.Init(&servers)

	lb := &LoadBalancer{
		port:    ":8080",
		servers: servers,
	}

	go lb.healthCheck()

	log.Printf("LoadBalancer listening on %s.. \n", lb.port)
	log.Fatal(http.ListenAndServe(lb.port, lb))
}
