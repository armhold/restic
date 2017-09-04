package web

import (
	"net/http"
	"fmt"
	"github.com/restic/restic/internal/restic"
	"sort"
	"context"
)

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("rootHandler\n")
	fmt.Printf("path: %q\n", r.URL.Path)

	// The "/" pattern matches everything, so we need to check that we're at the root here.
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

	currRepoName := r.FormValue("repo")

	cssClassForRepo := func(repoName string) (string) {
		// TODO: names might have spaces. Use id, or urlencode
		if repoName == r.FormValue(currRepoName) {
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
	}{
		Repos:     WebConfig.Repos,
		Flash:     flash,
		Css_class: cssClassForRepo,
	}

	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	repo, ok := findCurrRepoByName(currRepoName, WebConfig.Repos)
	if ok {
		err := listSnapshots(repo)
		if err != nil {
			fmt.Printf("listSnapshots: %s\n", err.Error())
		}
	}

	fmt.Printf("sucessful exit rootHandler()\n")
}

func findCurrRepoByName(name string, repos []Repo) (Repo, bool) {
	for _, r := range repos {
		if r.Name == name {
			fmt.Printf("found repo: %#v\n", r)
			return r, true
		}
	}

	return Repo{}, false
}

func listSnapshots(repo Repo) (error) {
	//r, err := OpenRepository("/Users/armhold/restic-web", "pass")
	r, err := OpenRepository(repo.Path, repo.Password)
	if err != nil {
		return err
	}

	list := restic.FindFilteredSnapshots(context.TODO(), r, "", []restic.TagList{}, []string{})
	sort.Sort(sort.Reverse(list))

	for _, s := range list {
		fmt.Printf("snapshot: %#v\n", s)
	}

	return err
}
