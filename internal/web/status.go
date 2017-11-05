package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

var (
	lpchan = make(chan chan *StatusUpdate)
)

func init() {
	//go runBackupSimulator()
	//go runRestoreSimulator()
}

// long-polling status updates

type StatusUpdate struct {
	RepoName      string `json:"RepoName"`
	PercentDone   int    `json:"PercentDone"`
	Type          string `json:"Type"` // "BACKUP" or "RESTORE"
	StatusMsg     string `json:"StatusMsg"`
	Indeterminate bool   `json:"Indeterminate"`
	Error         string `json:"Error"`
}

// do a long poll
func statusAjaxHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("statusAjaxHandler\n")

	timeout, err := strconv.Atoi(r.URL.Query().Get("timeout"))
	if err != nil || timeout > 180000 || timeout < 0 {
		timeout = 60000
	}

	fmt.Printf("statusAjaxHandler waiting for request...\n")

	myRequestChan := make(chan *StatusUpdate)

	select {

	// wait for producer
	case lpchan <- myRequestChan:

		// timeout
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		fmt.Printf("long poll request timed out\n")
		return
	}

	status := <-myRequestChan

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(status); err != nil {
		fmt.Printf("error encoding response %s\n", err)
	}

	fmt.Printf("sent update to client: %v\n", status)
}

func runBackupSimulator() {
	fmt.Printf("runBackupSimulator initial sleep...")
	time.Sleep(time.Second * 5)
	fmt.Printf("runBackupSimulator running...")

	// forever loop from 0..100
	for {
		for i := 0; i <= 100; i++ {
			status := StatusUpdate{Type: "BACKUP", RepoName: "local1", PercentDone: i}
			if i < 25 {
				status.Indeterminate = true
				status.StatusMsg = "scanning"
			} else {
				status.Indeterminate = false
				status.StatusMsg = "running"
			}

			if i%2 == 0 {
				status.PercentDone = i - 10
				status.RepoName = "local2"
			}
			UpdateStatus(status)

			fmt.Printf("runBackupSimulator: %d\n", i)
			time.Sleep(time.Millisecond * 200)
		}
	}
}

func runRestoreSimulator() {
	fmt.Printf("runRestoreSimulator initial sleep...")
	time.Sleep(time.Second * 5)
	fmt.Printf("runRestoreSimulator running...")

	for i := 0; i <= 100; i++ {
		status := StatusUpdate{Type: "RESTORE", RepoName: "local1", PercentDone: i, Indeterminate: true}
		if i < 25 {
			status.StatusMsg = "scanning"
		} else {
			status.StatusMsg = "running"
		}

		UpdateStatus(status)

		fmt.Printf("runRestoreSimulator: %d\n", i)
		time.Sleep(time.Millisecond * 500)
	}

	status := StatusUpdate{Type: "RESTORE", RepoName: "local1", Indeterminate: false}
	UpdateStatusBlocking(status)
}

func UpdateStatus(s StatusUpdate) {
	count := 0

Loop:
	// loop in case there are multiple clients waiting concurrently; we'll send the same status to each of them.
	// when no clients are waiting, then we break

	for {
		select {
		// if there's a client waiting
		case clientchan := <-lpchan:
			clientchan <- &s
			count++

		default:
			//prevent blocking if no clients available to consume the status
			break Loop
		}
	}

	fmt.Printf("handled a total of %d clients w status: %v\n", count, s)
}

// blocks the caller until a client consumes the status
func UpdateStatusBlocking(s StatusUpdate) {
	clientchan := <-lpchan
	clientchan <- &s

	fmt.Printf("returning from UpdateStatusBlocking\n")
}
