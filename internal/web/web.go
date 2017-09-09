package web

import (
	"fmt"
	"html/template"
	"net/http"
	"encoding/json"
)

var (
	// to pass FuncMap, order is important.
	// See: https://stackoverflow.com/questions/17843311/template-and-custom-function-panic-function-not-defined
	templates = template.Must(template.New("").Funcs(Helpers).ParseGlob("internal/web/*.html"))
	WebConfig Config
)

func init() {
}

func RunWeb(bindHost string, bindPort int) error {
	c, err := LoadConfigFromDefault()
	if err != nil {
		return err
	}

	WebConfig = c

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/addrepo", AddRepoAjaxHandler)
	http.HandleFunc("/addpath", AddDeletePathAjaxHandler)
	http.HandleFunc("/addexclude", AddDeleteExcludeAjaxHandler)
	http.HandleFunc("/snapshots", snapshotsHandler)
	http.HandleFunc("/paths", pathsHandler)
	http.HandleFunc("/excludes", excludeHandler)
	http.HandleFunc("/schedule", scheduleHandler)
	http.HandleFunc("/browse", browseHandler)

	// static assets
	fs := JustFilesFilesystem{http.Dir("assets")}
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(fs)))

	addr := fmt.Sprintf("%s:%d", bindHost, bindPort)

	fmt.Printf("binding to %s\n", addr)
	err = http.ListenAndServe(addr, nil)

	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	return err
}

func sendErrorToJs(w http.ResponseWriter, err string) {
	m := make(map[string]string)
	m["error"] = err

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
