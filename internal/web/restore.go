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

	isSelected := func(dir, path string) bool {
		fullPath := filepath.Join(dir, path)
		return repo.BackupPaths.Paths[fullPath]
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

	fmt.Printf("rendering...\n")

	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit navigateRestoreHandler\n")
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
			fmt.Printf("compare \"%s\" => \"%s\"\n", n.Name, d)

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
		fmt.Printf("\tcontents: %s\n", entry.Name)
	}

	return result, nil
}

// TODO: handle non-Unix paths
func splitIntoDirs(path string) []string {
	fmt.Printf("inside splitIntoDirs: \"%s\"\n", path)

	path = strings.Trim(path, string(filepath.Separator))
	fmt.Printf("after 1st trim, path is \"%s\"\n", path)

	if len(path) == 0 {
		return []string{}
	}

	return strings.Split(path, string(filepath.Separator))
}
