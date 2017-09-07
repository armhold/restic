package web

import (
	"fmt"
	"net/http"
	"os/user"
)

// browse the filesystem

func browseHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("browseHandler\n")

	flash, err := ParseFlashes(w, r)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	path := r.FormValue("path")
	if path == "" {
		fmt.Printf("no path given, starting with home dir\n")
		usr, err := user.Current()
		if err != nil {
			s := fmt.Sprintf("error getting current user: %s", err)
			flash.Danger += s
			fmt.Printf(s)
			path = "/"
		} else {
			path = usr.HomeDir
		}
	}


	fmt.Printf("list files in %s\n", path)


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
		Repos        []Repo
		CurrRepoName string
		Flash        Flash
		Css_class    func(repoName string) string
		Nav          *Navigation
		Tab          string
		currPath     string
	}{
		Repos:        WebConfig.Repos,
		CurrRepoName: currRepoName,
		Flash:        flash,
		Css_class:    cssClassForRepo,
		Nav:          &Navigation{req: r, Tab: "browse"},
		Tab:          "browse",
	}

	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit browseHandler()\n")
}
