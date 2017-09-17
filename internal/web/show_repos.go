package web

import (
	"context"
	"fmt"
	"github.com/restic/restic/internal/restic"
	"net/http"
	"sort"
)

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("rootHandler\n")

	// if we get here, none of the other handlers matched, so implement 404 behavior if client
	// not actually requesting root.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	flash, err := ParseFlashes(w, r)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// TODO: code repeated in web.go
	currRepoName := r.FormValue("repo")
	cssClassForRepo := func(repoName string) string {
		// TODO: names might have spaces. Use id, or urlencode
		if repoName == currRepoName {
			return "active"
		} else {
			return ""
		}
	}

	// if repos exist, choose one and redirect to snapshots tab
	//
	if len(WebConfig.Repos) > 1 {
		repo := r.FormValue("repo")
		if repo == "" {
			repo = WebConfig.Repos[0].Name
		}

		toUrl := SnapshotsUrl(repo)
		http.Redirect(w, r, toUrl, http.StatusSeeOther)
		fmt.Printf("redirecting from root to %s\n", toUrl)

		return
	}

	data := struct {
		Repos        []*Repo
		CurrRepoName string
		Flash        Flash
		Css_class    func(repoName string) string
		Nav          *Navigation
	}{
		Repos:        WebConfig.Repos,
		CurrRepoName: currRepoName,
		Flash:        flash,
		Css_class:    cssClassForRepo,
		Nav:          &Navigation{req: r, Tab: ""},
	}

	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit rootHandler()\n")
}

func findCurrRepoByName(name string, repos []*Repo) (*Repo, bool) {
	for _, r := range repos {
		if r.Name == name {
			fmt.Printf("found repo: %#v\n", r)
			return r, true
		}
	}

	return &Repo{}, false
}

func listSnapshots(repo *Repo) (restic.Snapshots, error) {
	var snaps restic.Snapshots

	r, err := OpenRepository(repo.Path, repo.Password)
	if err != nil {
		return snaps, err
	}

	snaps = restic.FindFilteredSnapshots(context.TODO(), r, "", []restic.TagList{}, []string{})
	sort.Sort(snaps)

	return snaps, nil
}
