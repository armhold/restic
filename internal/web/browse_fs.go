package web

import (
	"fmt"
	"net/http"
	"os/user"
	"io/ioutil"
	"github.com/restic/restic/internal/errors"
	"os"
	"path/filepath"
	"net/url"
	"sort"
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

	dir := r.FormValue("dir")
	if dir == "" {
		fmt.Printf("no dir given, starting with home dir\n")
		usr, err := user.Current()
		if err != nil {
			s := fmt.Sprintf("error getting current user: %s", err)
			flash.Danger += s
			fmt.Printf(s)
			dir = "/"
		} else {
			dir = usr.HomeDir
		}
	}

	fmt.Printf("list files in %s\n", dir)
	files, err := listDir(dir)
	if err != nil {
		flash.Danger += err.Error()
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

	nav := &Navigation{req: r, Tab: "browse"}

	linkToDir := func(file string) string {
		fullPath := filepath.Join(dir, file)
		return nav.BrowseUrl() + "&dir=" + url.QueryEscape(fullPath)
	}

	isDir := func(file string) bool {
		fullPath := filepath.Join(dir, file)
		s, err := os.Stat(fullPath)
		if err != nil {
			m := fmt.Sprintf("error getting file status for %s: %s\n", err)
			fmt.Printf(m)
			flash.Danger += m
		}

		return s.IsDir()
	}


	data := struct {
		Repos        []Repo
		CurrRepoName string
		Flash        Flash
		Css_class    func(repoName string) string
		Nav          *Navigation
		Tab          string
		Dir          string
		Files        []os.FileInfo
		LinkToDir    func(string) string
		IsDir        func(file string) bool
	}{
		Repos:        WebConfig.Repos,
		CurrRepoName: currRepoName,
		Flash:        flash,
		Css_class:    cssClassForRepo,
		Nav:          nav,
		Tab:          "browse",
		Dir:          dir,
		Files:        files,
		LinkToDir:    linkToDir,
		IsDir:        isDir,
	}

	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit browseHandler()\n")
}

func listDir(dir string) ([]os.FileInfo, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return files, errors.Errorf("error reading directory: %s", err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	return files, nil
}
