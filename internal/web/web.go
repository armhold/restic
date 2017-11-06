package web

import (
	"encoding/json"
	"fmt"
	"github.com/restic/restic/internal/repository"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
)

var (
	templates      *template.Template
	sessionManager Manager

	// sharedRepo to be accessed by handlers only via getRepo() and releaseRepo()
	sharedRepoMutex sync.Mutex
	sharedRepo      *repository.Repository
)

func init() {
	templatesDir := os.Getenv("GOPATH") + "/src/github.com/restic/restic/internal/web"
	path := filepath.Join(templatesDir, "*.html")
	fmt.Printf("attempting to load html templates from: %s\n", templatesDir)

	// to pass FuncMap, order is important.
	// See: https://stackoverflow.com/questions/17843311/template-and-custom-function-panic-function-not-defined
	templates = template.Must(template.New("").Funcs(Helpers).ParseGlob(path))
}

type FormErrors map[string]string

func getRepo() *repository.Repository {
	sharedRepoMutex.Lock()
	fmt.Printf("getRepo\n")

	return sharedRepo
}

func releaseRepo() {
	sharedRepoMutex.Unlock()

	fmt.Printf("releaseRepo\n")
}

// r should be a repository with index pre-loaded
func RunWeb(bindHost string, bindPort int, r *repository.Repository) error {
	sharedRepo = r

	http.HandleFunc("/", panicRecover(snapshotsHandler))
	http.HandleFunc("/status", panicRecover(statusAjaxHandler))
	http.HandleFunc("/nav", panicRecover(navigateRestoreHandler))
	//http.HandleFunc("/restore", panicRecover(doRestoreAjaxHandler))
	//http.HandleFunc("/deletesnapshot", panicRecover(deleteSnapshotAjaxHandler))
	//http.HandleFunc("/addrestorepath", panicRecover(addRemoveRestorePathAjaxHandler))
	//http.HandleFunc("/progress", panicRecover(inProgressHandler))
	//http.HandleFunc("/progress_poll", panicRecover(inProgressAjaxHandler))
	//http.HandleFunc("/stop_restore", panicRecover(stopRestoreAjaxHandler))

	// static assets
	fs := JustFilesFilesystem{http.Dir("assets")}
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(fs)))

	addr := fmt.Sprintf("%s:%d", bindHost, bindPort)

	fmt.Printf("binding to %s\n", addr)
	err := http.ListenAndServe(addr, nil)

	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	return err
}

// standard web server seems to swallow panics without logging them?
func panicRecover(f func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC RECOVERED: %s, %s\n", r, debug.Stack())
			}
		}()
		f(w, r)
	}
}

func sendErrorToJs(w http.ResponseWriter, err string) {
	m := make(map[string]string)
	m["error"] = err

	fmt.Println(err)
	sendErrorMapToJs(w, m)
}

func sendErrorMapToJs(w http.ResponseWriter, errMap map[string]string) {
	// javascript front-end expects errors as key/value pairs, so use a map
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	if err := json.NewEncoder(w).Encode(errMap); err != nil {
		fmt.Printf("error encoding response %s\n", err)
	}
}
