package web

import (
	"fmt"
	"net/http"
)


func init() {
}

func backupHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("backupHandler\n")

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
	if ! ok {
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

	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit backupHandler()\n")
}
