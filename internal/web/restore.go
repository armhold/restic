package web

import (
	"github.com/restic/restic/internal/restic"
	"path/filepath"
	"net/url"
	"net/http"
	"fmt"
	"strings"
	"time"
	"context"
	"github.com/restic/restic/internal/filter"
	"github.com/restic/restic/internal/errors"
	"sync"
)

func navigateRestoreHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("navigateRestoreHandler\n")

	flash, err := ParseFlashes(w, r)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	snapshotId := r.FormValue("snapshotId")
	if snapshotId == "" {
		fmt.Printf("no snapshotId given\n")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	dir := r.FormValue("dir")
	if dir == "" {
		fmt.Printf("no dir given, starting with root\n")
		dir = "/" // TODO: root for non-unix OSes
	}

	currRepoName := r.FormValue("repo")
	// TODO: code repeated in show_repos.go
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
		msg := fmt.Sprintf("error retrieving repo: %s", currRepoName)
		fmt.Println(msg)
		SaveFlashToCookie(w, "danger_flash", msg)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	fmt.Printf("list files in snapshot: %s under dir: %s\n", snapshotId, dir)
	files, err := listFilesUnderDirInSnapshot(repo, snapshotId, dir)
	if err != nil {
		fmt.Println(err)
		flash.Danger += err.Error()
	}

	nav := &Navigation{req: r, Tab: "restore"}

	linkToPath := func(path string) string {
		return fmt.Sprintf("/nav?repo=%s&amp;snapshotId=%s&amp;dir=%s", url.QueryEscape(currRepoName), url.QueryEscape(snapshotId), url.QueryEscape(path))
	}

	linkToFileInDir := func(file string) string {
		fullPath := filepath.Join(dir, file)
		return linkToPath(fullPath)
	}

	parentDir := filepath.Dir(dir)
	linkToParentDir := nav.BrowseUrl() + "&amp;dir=" + url.QueryEscape(parentDir)

	includedPaths := getIncludedPathsMapFromSession(w, r)

	isSelected := func(dir, path string) bool {
		fullPath := filepath.Join(dir, path)
		_, ok := includedPaths.Load(fullPath)
		return ok
	}

	// create links for drop-down for navigating to parent dirs
	// TODO: add volumename for all volumes on Windows
	// TODO: copied from browse.go
	var dirLinks []dirLink
	d := dir
	for d != filepath.VolumeName(d) && d != "/" {
		d = filepath.Dir(d);
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
		Files           []*snapshotPath
		LinkToFileInDir func(file string) string
		LinkToParentDir string
		IsSelected      func(dir, path string) bool
		ParentDirLinks  []dirLink
		SnapshotId      string
		SnapSelected    bool
	}{
		Repos:           WebConfig.Repos,
		CurrRepoName:    currRepoName,
		Flash:           flash,
		Css_class:       cssClassForRepo,
		Nav:             nav,
		Tab:             "restore", // NB: HTML does not have a restore tab yet
		Dir:             dir,
		Files:           files,
		LinkToFileInDir: linkToFileInDir,
		LinkToParentDir: linkToParentDir,
		IsSelected:      isSelected,
		ParentDirLinks:  dirLinks,
		SnapshotId:      snapshotId,
		SnapSelected:    true,
	}

	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit navigateRestoreHandler\n")
}

type restore struct {
	repo       string
	snapshotId string
	target     string
	path       string
}

func restoreFromForm(r *http.Request) restore {
	result := restore{}
	result.repo = r.FormValue("repo")
	result.snapshotId = r.FormValue("snapshotId")
	result.target = r.FormValue("target")
	result.path = r.FormValue("path")

	fmt.Printf("restore: %#v\n", result)

	return result
}

func (d *restore) Validate() (ok bool, errors FormErrors) {
	errors = make(map[string]string)

	if strings.TrimSpace(d.repo) == "" {
		errors["repo"] = "repository name missing"
	}

	if strings.TrimSpace(d.snapshotId) == "" {
		errors["snapshotId"] = "snapshot ID missing"
	}

	if strings.TrimSpace(d.target) == "" {
		errors["target"] = "target missing"
	}

	if strings.TrimSpace(d.path) == "" {
		errors["path"] = "path missing"
	}

	return len(errors) == 0, errors
}

func doRestoreHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("doRestoreHandler\n")

	restore := restoreFromForm(r)

	w.Header().Set("Content-Type", "application/json")

	ok, formErrors := restore.Validate()
	if ! ok {
		fmt.Println(formErrors)
		sendErrorMapToJs(w, formErrors)
		return
	}

	err, warnings := doRestore(restore)
	if err != nil {
		fmt.Println(err)
		sendErrorToJs(w, err.Error())
		return
	}

	if warnings > 0 {
		SaveFlashToCookie(w, "warn_flash", fmt.Sprintf("There were %d errors\n", warnings))
	}

	SaveFlashToCookie(w, "success_flash", fmt.Sprintf("Restore successful; files saved to %s", restore.target))

	w.WriteHeader(http.StatusOK)
	executeJs := fmt.Sprintf("{\"on_success\": \"window.location.href='/snapshots?repo=%s'\"}", restore.repo)
	w.Write([]byte(executeJs))

	fmt.Printf("sucessful exit doRestoreHandler\n")
}

// returns error (fatal if non-nil), and count of warnings. Warnings may occur e.g. setting ownership bits, etc.
func doRestore(restore restore) (error, int) {
	repo, ok := findCurrRepoByName(restore.repo, WebConfig.Repos)
	if ! ok {
		return errors.Errorf("error retrieving repo: %s", restore.repo), 0
	}

	fmt.Println("do restore: %#v\n", restore)

	repository, err := OpenRepository(repo.Path, repo.Password)
	if err != nil {
		return errors.Errorf("error opening repo: %s", err.Error()), 0
	}

	if err = repository.LoadIndex(context.TODO()); err != nil {
		return errors.Errorf("error loading index: %s", err.Error()), 0
	}

	snapshotID, err := restic.FindSnapshot(repository, restore.snapshotId)
	if err != nil {
		return errors.Errorf("invalid snapshot id: %q: %s", restore.snapshotId, err.Error()), 0
	}

	res, err := restic.NewRestorer(repository, snapshotID)
	if err != nil {
		return errors.Errorf("creating restorer failed: %v", err.Error), 0
	}

	warnings := 0
	res.Error = func(dir string, node *restic.Node, err error) error {
		fmt.Printf("ignoring error for %s: %s\n", dir, err)
		warnings++
		return nil
	}

	selectedPaths := []string{restore.path}

	res.SelectFilter = func(item string, dstpath string, node *restic.Node) (bool, bool) {
		matched, childMayMatch, err := filter.List(selectedPaths, item)
		if err != nil {
			fmt.Printf("error for path: %v", err)
		}

		return matched, childMayMatch
	}

	fmt.Printf("restoring %s to %s\n", res.Snapshot(), restore.target)

	err = res.RestoreTo(context.TODO(), restore.target)

	return err, warnings
}

// a path to be restored
type restoreEntry struct {
	dir      string
	name     string
	selected bool
}

func (e *restoreEntry) completePath() string {
	return filepath.Join(e.dir, e.name)
}

func restoreEntryFromForm(r *http.Request) restoreEntry {
	result := restoreEntry{}
	result.dir = r.FormValue("dir")
	result.name = r.FormValue("name")
	result.selected = r.FormValue("selected") == "true"

	return result
}

func (d *restoreEntry) Validate() (ok bool, errors FormErrors) {
	errors = make(map[string]string)

	if strings.TrimSpace(d.dir) == "" {
		errors["dir"] = "dir missing"
	}

	if strings.TrimSpace(d.name) == "" {
		errors["name"] = "name missing"
	}

	return len(errors) == 0, errors
}

// add/remove a path to/from the restore list when user clicks checkbox
func addRemoveRestorePathAjaxHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("enter addRemoveRestorePathAjaxHandler\n")

	err := r.ParseForm()
	if err != nil {
		fmt.Printf("error parsing form: %s\n", err.Error())
		return
	}

	entry := restoreEntryFromForm(r)
	ok, errMap := entry.Validate()
	if ! ok {
		sendErrorMapToJs(w, errMap)
		return
	}

	includedPaths := getIncludedPathsMapFromSession(w, r)

	if entry.selected {
		fmt.Printf("added: %s", entry.completePath())
		includedPaths.Store(entry.completePath(), true)
	} else {
		fmt.Printf("removed: %s", entry.completePath())
		includedPaths.Delete(entry.completePath())
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	executeJs := fmt.Sprintf("{\"on_success\": \"console.log('ok');\"}")
	w.Write([]byte(executeJs))

	fmt.Printf("successful exit addRemoveRestorePathAjaxHandler\n")
}

func getIncludedPathsMapFromSession(w http.ResponseWriter, r *http.Request) *sync.Map {
	// TODO: ok to set session cookie, and in the same request also send JSON response?
	session, _ := sessionManager.GetOrCreateSession(w, r)

	var includedPaths *sync.Map

	ip, ok := session.Get("included_paths")
	if ! ok {
		includedPaths = &sync.Map{}
		session.Set("included_paths", includedPaths)
		fmt.Printf("created new includedPaths map\n")
	} else {
		includedPaths = ip.(*sync.Map)
		fmt.Printf("use existing includedPaths map\n")
	}

	return includedPaths
}

type snapshotPath struct {
	Name  string
	IsDir bool
}

func listFilesUnderDirInSnapshot(repo *Repo, snapshotIDString, dir string) ([]*snapshotPath, error) {
	var result []*snapshotPath

	start := time.Now()

	r, err := OpenRepository(repo.Path, repo.Password)
	if err != nil {
		return result, err
	}
	fmt.Printf("OpenRepository took %s\n", time.Since(start))

	// TODO: lock repo here?

	start = time.Now()
	if err = r.LoadIndex(context.TODO()); err != nil {
		return result, err
	}
	fmt.Printf("LoadIndex took %s\n", time.Since(start))

	start = time.Now()
	snapshotID, err := restic.FindSnapshot(r, snapshotIDString)
	if err != nil {
		return result, fmt.Errorf("invalid id %q: %v", snapshotIDString, err)
	}
	fmt.Printf("FindSnapshot took %s\n", time.Since(start))

	start = time.Now()
	currSnapshot, err := restic.LoadSnapshot(context.TODO(), r, snapshotID)
	if err != nil {
		return result, fmt.Errorf("could not load snapshot %q: %v\n", snapshotID, err)
	}
	fmt.Printf("LoadSnapshot took %s\n", time.Since(start))

	dirs := splitIntoDirs(dir)

	fmt.Printf("dir is: \"%s\"\n", dir)

	tree, err := r.LoadTree(context.TODO(), *currSnapshot.Tree)
	if err != nil {
		return result, err
	}

	// walk the tree down to the current dir
	for _, d := range dirs {
		found := false
		for _, n := range tree.Nodes {
			if n.Type == "dir" && n.Subtree != nil && n.Name == d {
				start := time.Now()

				tree, err = r.LoadTree(context.TODO(), *n.Subtree)
				if err != nil {
					return result, err
				}
				elapsed := time.Since(start)
				fmt.Printf("LoadTree took %s\n", elapsed)

				found = true
				break
			}
		}

		if ! found {
			return result, fmt.Errorf("failed to find %s in snapshot", d)
		}
	}

	for _, entry := range tree.Nodes {
		result = append(result, &snapshotPath{Name: entry.Name, IsDir: entry.Type == "dir"})
	}

	return result, nil
}

// TODO: handle non-Unix paths
func splitIntoDirs(path string) []string {
	path = strings.Trim(path, string(filepath.Separator))

	if len(path) == 0 {
		return []string{}
	}

	return strings.Split(path, string(filepath.Separator))
}
