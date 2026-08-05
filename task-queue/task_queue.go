package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

//1. Job Interface

type Job interface {
	Execute() error
	GetRetries() int
	IncrementRetries()
	GetID() string
}

// 2. Concrete Job Type
type EmailJob struct {
	Recipient string
	Body      string
	Retries   int
}
type ImageProcessingJob struct {
	ImageURL string
	Retries  int
}

// Email PRocessing Unit
func (e EmailJob) Execute() error {
	fmt.Printf("Sending emial to %s .... (attempt #%d)\n", e.Recipient, e.Retries+1)
	time.Sleep(500 * time.Millisecond) // Simulating the network delay
	if rand.Intn(10) < 4 {
		return fmt.Errorf("Network timeout conneting to email server.....")
	}
	fmt.Printf("Email sent to %s successfully!\n", e.Recipient)
	return nil
}
func (e EmailJob) GetRetries() int    { return e.Retries }
func (e *EmailJob) IncrementRetries() { e.Retries++ }
func (e EmailJob) GetID() string      { return "Email:" + e.Recipient }

// Image Prcoessing unit
func (I ImageProcessingJob) Execute() error {
	fmt.Printf("Downloading image from url : %s ....(attempt #%d)\n", I.ImageURL, I.Retries+1)
	time.Sleep(500 * time.Millisecond) // Simulating the network delay
	if rand.Intn(10) < 5 {
		return fmt.Errorf("Network timeout connecting to image server")
	}
	fmt.Printf("Image Downloaded Successfully from %s successfully!\n", I.ImageURL)
	return nil
}
func (e ImageProcessingJob) GetRetries() int    { return e.Retries }
func (e *ImageProcessingJob) IncrementRetries() { e.Retries++ }
func (e ImageProcessingJob) GetID() string      { return "Image:" + e.ImageURL }

// 3. The Task Queue Engine
type WorkerPool struct {
	workerCount int
	jobQueue    chan Job
	wg          sync.WaitGroup
	maxRetries  int
}

// NewWorkerPool initializes the pool
func NewWorkerPool(workers int, queueSize int) *WorkerPool {
	return &WorkerPool{
		workerCount: workers,
		jobQueue:    make(chan Job, queueSize),
		maxRetries:  3,
	}
}

// Start boots up the background workers.
func (p *WorkerPool) Start() {
	fmt.Printf("Starting worker pool with %d workers.. \n", p.workerCount)

	for i := 0; i < p.workerCount; i++ {
		go func(workerID int) {
			for job := range p.jobQueue {
				fmt.Printf("[Worker %d] Processing job...\n", workerID)
				if err := job.Execute(); err != nil {
					fmt.Printf("[Worker %d] Job failed: %v\n", workerID, err)

					if job.GetRetries() < p.maxRetries {
						job.IncrementRetries()
						fmt.Printf("[Worker %d] 🔄 Re-queuing job: %s (retry %d/%d)\n", workerID, job.GetID(), job.GetRetries(), p.maxRetries)
						p.AddJob(job)
					} else {
						fmt.Printf("[Worker %d] 💀 Job dead-lettered: %s\n", workerID, job.GetID())
					}
				}
				p.wg.Done()
			}
		}(i)
	}
}

// AddJob pushes a new task into the queue.
func (p *WorkerPool) AddJob(j Job) {
	p.wg.Add(1)
	p.jobQueue <- j
}

// Stop waits for all jobs to finish and cleanly shuts down the pool.
func (p *WorkerPool) Stop() {
	// TODO: Your code goes here!
	// Hint: Close the channel to tell workers no more jobs are coming.
	// Then wait for the WaitGroup to reach zero.
	p.wg.Wait()
	close(p.jobQueue)

	fmt.Println("All jobs processed. Shutting down pool.")
}

func main() {

	// --- Email Pool (no retries needed, always works) ---
	emailPool := NewWorkerPool(2, 20)
	emailPool.Start()

	emails := []string{"alice@x.com", "bob@x.com", "charlie@x.com"}
	for _, email := range emails {
		emailPool.AddJob(&EmailJob{Recipient: email, Body: "Hello!", Retries: 0})
	}
	emailPool.Stop()

	fmt.Println("\n" + "═══════════════════════════════════════" + "\n")

	// --- Image Pool (with retries, flaky network) ---
	imagePool := NewWorkerPool(3, 30) // bigger queue for retries
	imagePool.Start()

	imageURLs := []string{
		"img1.jpg", "img2.jpg", "img3.jpg", "img4.jpg", "img5.jpg",
		"img6.jpg", "img7.jpg", "img8.jpg",
	}
	for _, url := range imageURLs {
		imagePool.AddJob(&ImageProcessingJob{ImageURL: url, Retries: 0})
	}
	imagePool.Stop()
}
