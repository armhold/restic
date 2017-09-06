package web

import (
	"net/http"
	"fmt"
	"html/template"
	"github.com/restic/restic/internal/restic"
)

var (
	// to pass FuncMap, order is important.
	// See: https://stackoverflow.com/questions/17843311/template-and-custom-function-panic-function-not-defined
	templates = template.Must(template.New("").Funcs(Helpers).ParseGlob("internal/web/*.html"))
	WebConfig Config
)

func init() {
}

type Repo struct {
	Name     string `json:"Name"`     // "local repo"
	Path     string `json:"Path"`     //  "b2:bucket-Name/Path"
	Password string `json:"Password"` // TODO: encrypt?
}

func RunWeb(bindHost string, bindPort int) error {
	c, err := LoadConfigFromDefault()
	if err != nil {
		return err
	}

	WebConfig = c

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/addrepo", AddRepoAjaxHandler)
	http.HandleFunc("/snapshots", snapshotsHandler)
	http.HandleFunc("/paths", pathsHandler)
	http.HandleFunc("/excludes", excludeHandler)
	http.HandleFunc("/schedule", scheduleHandler)

	// static assets
	fs := JustFilesFilesystem{http.Dir("assets")}
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(fs)))

	addr := fmt.Sprintf("%s:%d", bindHost, bindPort)

	fmt.Printf("binding to %s\n", addr)
	err = http.ListenAndServe(addr, nil)
	return err
}

func snapshotsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("snapshotsHandler\n")
	fmt.Printf("path: %q\n", r.URL.Path)

	flash, err := ParseFlashes(w, r)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// TODO: code repeated in show_repos.go
	currRepoName := r.FormValue("repo")
	cssClassForRepo := func(repoName string) (string) {
		// TODO: names might have spaces. Use id, or urlencode
		if repoName == currRepoName {
			return "active"
		} else {
			return ""
		}
	}

	var snaps restic.Snapshots
	repo, ok := findCurrRepoByName(currRepoName, WebConfig.Repos)
	if ok {
		snaps, err = listSnapshots(repo)
		if err != nil {
			fmt.Printf("listSnapshots: %s\n", err.Error())

		}
	}

	data := struct {
		Repos        []Repo
		CurrRepoName string
		Flash        Flash
		Css_class    func(repoName string) (string)
		Snapshots    restic.Snapshots
		Nav          *Navigation
		Tab          string
	}{
		Repos:     WebConfig.Repos,
		Flash:     flash,
		Css_class: cssClassForRepo,
		Snapshots: snaps,
		Nav:       &Navigation{req: r},
		Tab:       "snapshots",
	}

	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit snapshotsHandler()\n")
}

func pathsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("pathsHandler\n")

	flash, err := ParseFlashes(w, r)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// TODO: code repeated in show_repos.go
	currRepoName := r.FormValue("repo")
	cssClassForRepo := func(repoName string) (string) {
		// TODO: names might have spaces. Use id, or urlencode
		if repoName == currRepoName {
			return "active"
		} else {
			return ""
		}
	}

	data := struct {
		Repos        []Repo
		CurrRepoName string
		Flash        Flash
		Css_class    func(repoName string) (string)
		Nav          *Navigation
		Tab          string
	}{
		Repos:     WebConfig.Repos,
		Flash:     flash,
		Css_class: cssClassForRepo,
		Nav:       &Navigation{req: r},
		Tab:       "paths",
	}

	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit pathsHandler()\n")
}

func excludeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("excludeHandler\n")

	flash, err := ParseFlashes(w, r)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// TODO: code repeated in show_repos.go
	currRepoName := r.FormValue("repo")
	cssClassForRepo := func(repoName string) (string) {
		// TODO: names might have spaces. Use id, or urlencode
		if repoName == currRepoName {
			return "active"
		} else {
			return ""
		}
	}

	data := struct {
		Repos        []Repo
		CurrRepoName string
		Flash        Flash
		Css_class    func(repoName string) (string)
		Nav          *Navigation
		Tab          string
	}{
		Repos:     WebConfig.Repos,
		Flash:     flash,
		Css_class: cssClassForRepo,
		Nav:       &Navigation{req: r},
		Tab:       "excludes",
	}

	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit excludesHandler()\n")
}

func scheduleHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("scheduleHandler\n")
}
