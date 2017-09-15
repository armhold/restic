package web

import (
	"github.com/restic/restic/internal/restic"
	"net/http"
	"fmt"
)

func restoreHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("restoreHandler\n")

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

	var snaps restic.Snapshots
	repo, ok := findCurrRepoByName(currRepoName, WebConfig.Repos)
	if ok {
		snaps, err = listSnapshots(repo)
		if err != nil {
			fmt.Printf("listSnapshots: %s\n", err.Error())

			// NB: don't call SaveFlashToCookie() because we want it to render immediately here, not after redirect
			flash.Danger += fmt.Sprintf("error listing snapshots: %s", err)
		}
	}

	data := struct {
		Repos        []*Repo
		CurrRepoName string
		Flash        Flash
		Css_class    func(repoName string) string
		Snapshots    restic.Snapshots
		Nav          *Navigation
	}{
		Repos:     WebConfig.Repos,
		CurrRepoName: currRepoName,
		Flash:     flash,
		Css_class: cssClassForRepo,
		Snapshots: snaps,
		Nav:       &Navigation{req: r, Tab: "restore"},
	}

	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit restoreHandler()\n")
}
