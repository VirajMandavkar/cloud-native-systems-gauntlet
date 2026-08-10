package main

import (
	"fmt"
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

func Worker(Service ServiceType, Log chan LogEntry) {
	for {
		now := time.Now()
		logs := LogEntry{
			ServiceData: Service,
			Status:      "Server is running normally!",
			Time:        &now,
		}

		Log <- logs
		time.Sleep(3 * time.Second)
	}
}

func Writer(Log chan LogEntry) {
	for entry := range Log {
		fmt.Printf("Service: %s | %v , Status: %s, Time: %v \n", entry.ServiceData.Name, entry.ServiceData.ID, entry.Status, entry.Time.String())
	}
}

func main() {

	services := ServiceType{
		Name: "Apache",
		ID:   1,
	}
	myPipe := make(chan LogEntry, 1000)

	go Worker(services, myPipe)
	Writer(myPipe)

	return
}
