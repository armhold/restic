package web

import (
	"net/http"
	"fmt"
	"sync"
	"context"
	"github.com/restic/restic/internal/errors"
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

func GetRestoreInProgressInstance() *RestoreInProgress {
	once.Do(func() {
		instance = &RestoreInProgress{}
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

	r.Ctx, r.CancelFunc = context.WithCancel(context.TODO())
	return r.Ctx, nil
}

func (r *RestoreInProgress) End() {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.running = false
}

func inProgressHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("inProgressHandler\n")

	flash, err := ParseFlashes(w, r)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// TODO: code repeated in show_repos.go
	currRepoName := r.FormValue("repo")
	cssClassForRepo := func(repoName string) string {
		// TODO: names might have spaces. Use id, or urlencode
		if repoName == currRepoName {
			return "active"
		} else {
			return ""
		}
	}

	repo, ok := findCurrRepoByName(currRepoName, WebConfig.Repos)
	if !ok {
		// NB: don't call SaveFlashToCookie() because we want it to render immediately here, not after redirect
		flash.Danger += fmt.Sprintf("error retrieving repo: %s", currRepoName)
	}

	data := struct {
		Repos        []*Repo
		CurrRepoName string
		Flash        Flash
		Css_class    func(repoName string) string
		Nav          *Navigation
		Paths        []string
	}{
		Repos:        WebConfig.Repos,
		CurrRepoName: currRepoName,
		Flash:        flash,
		Css_class:    cssClassForRepo,
		Nav:          &Navigation{req: r, Tab: "backup"},
		Paths:        sortedPaths(repo),
	}

	if err := templates.ExecuteTemplate(w, "in_progress.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit inProgressHandler()\n")
}

func inProgressAjaxHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("inProgressAjaxHandler\n")

	err := r.ParseForm()
	if err != nil {
		fmt.Printf("error parsing form: %s\n", err.Error())
		return
	}

	currRepoName := r.FormValue("repo")
	repo, ok := findCurrRepoByName(currRepoName, WebConfig.Repos)

	w.Header().Set("Content-Type", "application/json")

	if !ok {
		sendErrorToJs(w, fmt.Sprintf("could not find repo: %s", currRepoName))
		return
	}

	go func() {
		err := runBackup(repo)

		bs := BackupStatus{RepoName: currRepoName, PercentDone: 100}

		if err != nil {
			bs.Error = fmt.Sprintf("%s: backup failed: %s", currRepoName, err.Error())
		} else {
			bs.StatusMsg = fmt.Sprintf("%s: backup complete", currRepoName)
		}

		UpdateStatusBlocking(bs)
	}()

	w.WriteHeader(http.StatusOK)
	executeJs := fmt.Sprintf("{\"on_success\": \"backup started for %s\"}", currRepoName)
	w.Write([]byte(executeJs))
}
