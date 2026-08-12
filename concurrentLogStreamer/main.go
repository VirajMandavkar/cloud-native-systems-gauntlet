package main

import (
	"context"
	"encoding/json"
	"fmt"

	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

type LogEntry struct {
	ServiceData ServiceType `json:"service"` // containing the server id and service name
	Status      string      `json:"status"`  // containing the error
	Time        *time.Time  `json:"time,omitempty"`
}

type ServiceType struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

func logHandler(Log chan LogEntry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var entry LogEntry
		err := json.NewDecoder(r.Body).Decode(&entry)
		if err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		now := time.Now()
		entry.Time = &now

		Log <- entry

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Log received\n"))
	}
}

func Writer(Log chan LogEntry, doneChan chan bool) {

	currentDate := time.Now().Format("02-01-2006")
	fileName := fmt.Sprintf("%s-error-logs.log", currentDate)

	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer func() { file.Close() }()

	for entry := range Log {

		logLine := fmt.Sprintf("Service: %s | %v , Status: %s, Time: %v \n",
			entry.ServiceData.Name,
			entry.ServiceData.ID,
			entry.Status,
			entry.Time.Format("15:04:05.000"),
		)

		if strings.Contains(entry.Status, "ERROR") {
			today := time.Now().Format("02-01-2006")

			//Check for rotation
			if today != currentDate {
				fmt.Println("Rotating lof file....")
				file.Close()

				currentDate = today
				fileName = fmt.Sprintf("%s-error-logs.log", currentDate)

				file, err = os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					panic(err)
				}
				file.WriteString("----LOG CREATED AT " + time.Now().String() + " ----\n")
			}

			file.WriteString(logLine)
			fmt.Print(logLine)

		} else {
			fmt.Print(logLine)
		}
	}
	file.Close()
	close(doneChan)
}

func main() {

	myPipe := make(chan LogEntry, 1000)
	doneChan := make(chan bool)

	go Writer(myPipe, doneChan)

	http.HandleFunc("/logs", logHandler(myPipe))
	server := &http.Server{
		Addr: ":8080",
	}

	go func() {
		fmt.Println("[+] Server listening on port 8080....")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	<-sigChan
	fmt.Printf("\n[i] Shutdown signal recieved. Stopping network traffic.....\n")

	server.Shutdown(context.Background())

	close(myPipe)

	<-doneChan
	fmt.Println("[✓] Shutdown complete, Safe to exit.")

}
