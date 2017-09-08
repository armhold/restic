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
	"encoding/json"
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

	linkToFileInDir := func(file string) string {
		fullPath := filepath.Join(dir, file)
		return nav.BrowseUrl() + "&amp;dir=" + url.QueryEscape(fullPath)
	}

	parentDir := filepath.Dir(dir)
	linkToParentDir := nav.BrowseUrl() + "&amp;dir=" + url.QueryEscape(parentDir)

	data := struct {
		Repos           []*Repo
		CurrRepoName    string
		Flash           Flash
		Css_class       func(repoName string) string
		Nav             *Navigation
		Tab             string
		Dir             string
		Files           []os.FileInfo
		LinkToFileInDir func(string) string
		LinkToParentDir string
	}{
		Repos:           WebConfig.Repos,
		CurrRepoName:    currRepoName,
		Flash:           flash,
		Css_class:       cssClassForRepo,
		Nav:             nav,
		Tab:             "browse",
		Dir:             dir,
		Files:           files,
		LinkToFileInDir: linkToFileInDir,
		LinkToParentDir: linkToParentDir,
	}

	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit browseHandler()\n")
}

// add a new path to the backup list
func AddPathAjaxHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Printf("error parsing form: %s\n", err.Error())
		return
	}

	path := r.FormValue("path")
	dir := r.FormValue("dir")
	selected := r.FormValue("selected") == "true"
	repoName := r.FormValue("repo")

	fmt.Printf("update repo: \"%s\"\n" , repoName)
	fmt.Printf("\tdir: \"%s\"\n", dir)
	fmt.Printf("\tpath: \"%s\"\n", path)
	fmt.Printf("\tselected: %t\n", selected)

	var repo *Repo
	for _, r := range WebConfig.Repos {
		if r.Name == repoName {
			repo = r
			break
		}
	}

	if r == nil {
		errors := errors.Errorf("could not find repo: %s", repoName)
		fmt.Printf("AddPathAjaxHandler validation failed: %v\n", errors)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		errAsString, err := json.Marshal(errors)
		if err != nil {
			fmt.Printf("json.Marshal err: %s\n", err)
			return
		} else {
			fmt.Printf("sending errs: %s\n", errAsString)
		}

		if err := json.NewEncoder(w).Encode(errors); err != nil {
			fmt.Printf("error encoding response %s\n", err)
			return
		}

		return
	}

	// TODO: mutex on WebConfig before we modify it
	repo.BackupPaths.Paths[path] = true
	err = WebConfig.Save()

	// NB: order seems to matter here.
	// 1) content-type
	// 2) cookies
	// 3) status
	// 4) JSON response

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		fmt.Printf("error saving config: %s\n", err)
		SaveFlashToCookie(w, "danger_flash", fmt.Sprintf("Error adding new path: %s", err))
	} else {
		SaveFlashToCookie(w, "success_flash", fmt.Sprintf("New path \"%s\" added", path))
	}

	w.WriteHeader(http.StatusOK)

	executeJs := fmt.Sprintf("{\"on_success\": \"console.log('ok');\"}")
	w.Write([]byte(executeJs))
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
