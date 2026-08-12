package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

type LogEntry struct {
	ServiceData ServiceType // containing the server id and service name
	Status      string      // containing the error
	Time        *time.Time
}

type ServiceType struct {
	Name string
	ID   int
}

func Worker(Service ServiceType, Log chan LogEntry, quit chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	StatusType := []string{"INFO : All good !", "WARN : High CPU", "ERROR : Connection Loss"}
	for {
		randomInt := rand.IntN(3)
		sleepTime := time.Duration(randomInt+1) * time.Second
		select {
		case <-quit:
			fmt.Printf("Worker %s shutting down...\n", Service.Name)
			return

		case <-time.After(sleepTime):
			now := time.Now()
			logs := LogEntry{
				ServiceData: Service,
				Status:      StatusType[randomInt],
				Time:        &now,
			}
			Log <- logs
		}
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

	services := []ServiceType{
		{Name: "Apache", ID: 1},
		{Name: "Web1", ID: 2},
		{Name: "Web2", ID: 3},
	}

	myPipe := make(chan LogEntry, 1000)
	quitChan := make(chan bool)
	doneChan := make(chan bool)

	var wg sync.WaitGroup

	for _, svc := range services {
		wg.Add(1)
		go Worker(svc, myPipe, quitChan, &wg)
	}

	go Writer(myPipe, doneChan)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	<-sigChan
	fmt.Printf("\n[i] Shutdown signal recieved. Stopping workers.....\n")

	close(quitChan)

	wg.Wait()

	close(myPipe)

	<-doneChan
	fmt.Println("[✓] Shutdown complete, Safe to exit.")

}
