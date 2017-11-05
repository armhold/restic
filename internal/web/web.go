package web

import (
	"encoding/json"
	"fmt"
	"github.com/restic/restic/internal/restic"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
)

var (
	WebConfig      Config
	templates      *template.Template
	sessionManager Manager
)

func init() {
	templatesDir := os.Getenv("GOPATH") + "/src/github.com/restic/restic/internal/web"
	path := filepath.Join(templatesDir, "*.html")
	fmt.Printf("attempting to load html templates from: %s\n", templatesDir)

	// to pass FuncMap, order is important.
	// See: https://stackoverflow.com/questions/17843311/template-and-custom-function-panic-function-not-defined
	templates = template.Must(template.New("").Funcs(Helpers).ParseGlob(path))
}

func RunWeb(bindHost string, bindPort int, repo restic.Repository) error {
	http.HandleFunc("/", panicRecover(snapshotsHandler))
	//http.HandleFunc("/addrepo", panicRecover(addRepoAjaxHandler))
	//http.HandleFunc("/addpath", panicRecover(addDeletePathAjaxHandler))
	//http.HandleFunc("/addexclude", panicRecover(addDeleteExcludeAjaxHandler))
	//http.HandleFunc("/snapshots", panicRecover(snapshotsHandler))
	//http.HandleFunc("/paths", panicRecover(pathsHandler))
	//http.HandleFunc("/excludes", panicRecover(excludeHandler))
	//http.HandleFunc("/schedule", panicRecover(scheduleHandler))
	//http.HandleFunc("/backup", panicRecover(backupHandler))
	//http.HandleFunc("/browse", panicRecover(browseHandler))
	//http.HandleFunc("/runbackup", panicRecover(runBackupAjaxHandler))
	//http.HandleFunc("/status", panicRecover(statusAjaxHandler))
	//http.HandleFunc("/nav", panicRecover(navigateRestoreHandler))
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
