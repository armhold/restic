package web

import (
	"net/http"
	"fmt"
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

	cssClassForRepo := func(repoName string) (string) {
		// TODO: names might have spaces. Use id, or urlencode
		if repoName == r.FormValue("repo") {
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

	fmt.Printf("sucessful exit rootHandler()\n")
}
