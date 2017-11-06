package web

import (
	"context"
	"fmt"
	"github.com/restic/restic/internal/errors"
	"net/http"
	"sync"
)

var (
	instance *RestoreInProgress
	once     sync.Once
)

// 1. for simplicity, allow only a single restore to be active at a given time.
// 2. allow restore to be canceled asynchronously by http requests.
type RestoreInProgress struct {
	running    bool
	Ctx        context.Context
	CancelFunc context.CancelFunc
	lock       sync.Mutex
}

func noRestoreRunning() {
	fmt.Printf("no restore runnong, nothing to cancel\n")
}

func GetRestoreInProgressInstance() *RestoreInProgress {
	once.Do(func() {
		instance = &RestoreInProgress{CancelFunc: noRestoreRunning}
	})
	return instance
}

func (r *RestoreInProgress) Begin() (context.Context, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.running {
		return nil, errors.New("restore already in progress")
	}

	r.running = true

	ctx, cancel := context.WithCancel(context.TODO())

	r.Ctx = ctx
	r.CancelFunc = func() {
		fmt.Printf("Cancelling restore...\n")
		cancel()
	}

	return r.Ctx, nil
}

func (r *RestoreInProgress) End() {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.running = false
	r.CancelFunc = noRestoreRunning
}

func inProgressHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("inProgressHandler\n")

	flash, err := ParseFlashes(w, r)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	data := struct {
		Repos        []*Repo
		CurrRepoName string
		Flash        Flash
		Css_class    func(repoName string) string
		Nav          *Navigation
		Paths        []string
	}{
		Repos: WebConfig.Repos,
		Flash: flash,
		Nav:   &Navigation{req: r, Tab: "backup"},
	}

	if err := templates.ExecuteTemplate(w, "in_progress.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("successful exit inProgressHandler()\n")
}

func inProgressAjaxHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("inProgressAjaxHandler\n")

	err := r.ParseForm()
	if err != nil {
		fmt.Printf("error parsing form: %s\n", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	executeJs := fmt.Sprintf("{\"on_success\": \"console.log('ok');\"}")
	w.Write([]byte(executeJs))
}

func stopRestoreAjaxHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("stopRestoreAjaxHandler\n")

	err := r.ParseForm()
	if err != nil {
		fmt.Printf("error parsing form: %s\n", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")

	GetRestoreInProgressInstance().CancelFunc()

	w.WriteHeader(http.StatusOK)
	executeJs := fmt.Sprintf("{\"on_success\": \"console.log('ok');\"}")
	w.Write([]byte(executeJs))
}
