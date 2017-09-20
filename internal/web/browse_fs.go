package web

import (
	"fmt"
	"github.com/restic/restic/internal/errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"sort"
)

// browse the filesystem

// for rendering parent dir links in template dropdown
type dirLink struct {
	Dir  string
	Link string
}

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

	linkToPath := func(path string) string {
		return nav.BrowseUrl() + "&amp;dir=" + url.QueryEscape(path)
	}

	linkToFileInDir := func(file string) string {
		fullPath := filepath.Join(dir, file)
		return linkToPath(fullPath)
	}

	parentDir := filepath.Dir(dir)
	linkToParentDir := nav.BrowseUrl() + "&amp;dir=" + url.QueryEscape(parentDir)

	repo, ok := findCurrRepoByName(currRepoName, WebConfig.Repos)
	if !ok {
		// NB: don't call SaveFlashToCookie() because we want it to render immediately here, not after redirect
		flash.Danger += fmt.Sprintf("error retrieving repo: %s", currRepoName)
	} else {

	}

	isSelected := func(dir, path string) bool {
		fullPath := filepath.Join(dir, path)
		return repo.BackupPaths.Paths[fullPath]
	}

	// create links for drop-down for navigating to parent dirs
	// TODO: add volumename for all volumes on Windows
	var dirLinks []dirLink
	d := dir
	for d != filepath.VolumeName(d) && d != "/" {
		d = filepath.Dir(d)
		dl := dirLink{Dir: d, Link: linkToPath(d)}
		dirLinks = append(dirLinks, dl)
	}

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
		IsSelected      func(dir, path string) bool
		ParentDirLinks  []dirLink
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
		IsSelected:      isSelected,
		ParentDirLinks:  dirLinks,
	}

	if err := templates.ExecuteTemplate(w, "browse.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit browseHandler()\n")
}

// add/remove path to/from the backup list
// when method=DELETE, the path is deleted
func addDeletePathAjaxHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Printf("error parsing form: %s\n", err.Error())
		return
	}

	path := r.FormValue("path")
	dir := r.FormValue("dir")
	createPath := r.Method != "DELETE"
	repoName := r.FormValue("repo")
	fullpath := filepath.Join(dir, path)

	// if adding a path, ensure it exists
	if createPath {
		if _, err := os.Stat(fullpath); os.IsNotExist(err) {
			// path/to/whatever does not exist
			sendErrorToJs(w, fmt.Sprintf("path does not exist: %s", fullpath))
			return
		}
	}

	fmt.Printf("update repo: \"%s\"\n", repoName)
	fmt.Printf("\tmethod: \"%s\"\n", r.Method)
	fmt.Printf("\tdir: \"%s\"\n", dir)
	fmt.Printf("\tpath: \"%s\"\n", path)
	fmt.Printf("\tfullpath: \"%s\"\n", fullpath)
	fmt.Printf("\tcreatePath: %t\n", createPath)

	currRepoName := r.FormValue("repo")
	repo, ok := findCurrRepoByName(currRepoName, WebConfig.Repos)

	if !ok {
		sendErrorToJs(w, fmt.Sprintf("could not find repo: %s", repoName))
		return
	}

	// TODO: mutex on WebConfig before we modify it
	if createPath {
		repo.BackupPaths.Paths[fullpath] = true
	} else {
		delete(repo.BackupPaths.Paths, fullpath)
	}
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
