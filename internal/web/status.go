package web

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	"encoding/json"
)

var (
	lpchan = make(chan chan *BackupStatus)
)

func init() {
	go runProducer()
}

// long-polling status updates

type BackupStatus struct {
	RepoName    string `json:"RepoName"`
	PercentDone int    `json:"PercentDone"`
	StatusMsg   string `json:"StatusMsg"`
}

// do a long poll
func StatusAjaxHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("StatusAjaxHandler\n")

	timeout, err := strconv.Atoi(r.URL.Query().Get("timeout"))
	if err != nil || timeout > 180000 || timeout < 0 {
		timeout = 60000
	}

	fmt.Printf("StatusAjaxHandler waiting for request...\n")

	myRequestChan := make(chan *BackupStatus)

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

func runProducer() {
	fmt.Printf("runProducer initial sleep...")
	time.Sleep(time.Second * 5)
	fmt.Printf("runProducer running...")

	// forever loop from 0..100
	for {
		for i := 0; i <= 100; i++ {
		//Loop:
		//	for {
				select {
				// if there's a client waiting
				case clientchan := <-lpchan:
					status := &BackupStatus{RepoName: "local1", PercentDone: i, StatusMsg: "running..."}
					clientchan <- status
				default:
					//break Loop
				}
			//}

			fmt.Printf("runProducer: %d\n", i)
			time.Sleep(time.Millisecond * 200)
		}
	}
}
