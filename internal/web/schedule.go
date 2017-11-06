package web

import (
	"fmt"
	"net/http"
)

func scheduleHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("scheduleHandler\n")

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

	data := struct {
		Repos        []*Repo
		CurrRepoName string
		Flash        Flash
		Css_class    func(repoName string) string
		Nav          *Navigation
		Tab          string
	}{
		Repos:        WebConfig.Repos,
		CurrRepoName: currRepoName,
		Flash:        flash,
		Css_class:    cssClassForRepo,
		Nav:          &Navigation{req: r, Tab: "schedule"},
	}

	if err := templates.ExecuteTemplate(w, "schedule.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("successful exit scheduleHandler()\n")
}
