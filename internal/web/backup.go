package web

import (
	"fmt"
	"net/http"
	"path/filepath"
	"github.com/restic/restic/internal/archiver"
	"github.com/restic/restic/internal/restic"
	"github.com/restic/restic/internal/errors"
	"os"
	"github.com/restic/restic/internal/fs"
	"context"
	"github.com/restic/restic/internal/filter"
	"time"
	"strings"
)


func init() {
}

func backupHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("backupHandler\n")

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

	repo, ok := findCurrRepoByName(currRepoName, WebConfig.Repos)
	if ! ok {
		// NB: don't call SaveFlashToCookie() because we want it to render immediately here, not after redirect
		flash.Danger += fmt.Sprintf("error retrieving repo: %s", currRepoName)
	}

	data := struct {
		Repos        []*Repo
		CurrRepoName string
		Flash        Flash
		Css_class    func(repoName string) string
		Nav          *Navigation
		Paths        []string
	}{
		Repos:        WebConfig.Repos,
		CurrRepoName: currRepoName,
		Flash:        flash,
		Css_class:    cssClassForRepo,
		Nav:          &Navigation{req: r, Tab: "backup"},
		Paths:        sortedPaths(repo),
	}

	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit backupHandler()\n")
}


func runBackup(r Repo) error {
	target := r.BackupPaths.GetPaths()

	target, err := filterExisting(target)
	if err != nil {
		return err
	}

	// allowed devices
	ExcludeOtherFS := false
	var allowedDevs map[string]uint64
	if ExcludeOtherFS {
		allowedDevs, err = gatherDevices(target)
		if err != nil {
			return err
		}
		fmt.Printf("allowed devices: %v\n", allowedDevs)
	}

	repo, err := OpenRepository(r.Path, r.Password)
	if err != nil {
		return err
	}

	lock, err := lockRepo(repo)
	defer unlockRepo(lock)
	if err != nil {
		return err
	}

	err = repo.LoadIndex(context.TODO())
	if err != nil {
		return err
	}

	var parentSnapshotID *restic.ID

	// Find last snapshot to set it as parent, if not already set
	var tagLists []restic.TagList
	hostname := ""
	id, err := restic.FindLatestSnapshot(context.TODO(), repo, target, tagLists, hostname)
	if err == nil {
		parentSnapshotID = &id
	} else if err != restic.ErrNoSnapshotFound {
		return err
	}

	if parentSnapshotID != nil {
		fmt.Printf("using parent snapshot %v\n", parentSnapshotID.Str())
	}

	fmt.Printf("scan %v\n", target)

	excludes := r.BackupPaths.GetExcludes()

	selectFilter := func(item string, fi os.FileInfo) bool {
		matched, _, err := filter.List(excludes, item)
		if err != nil {
			fmt.Printf("error for exclude pattern: %v", err)
		}

		if matched {
			fmt.Printf("path %q excluded by a filter", item)
			return false
		}

		if !ExcludeOtherFS || fi == nil {
			return true
		}

		id, err := fs.DeviceID(fi)
		if err != nil {
			// This should never happen because gatherDevices() would have
			// errored out earlier. If it still does that's a reason to panic.
			panic(err)
		}

		for dir := item; dir != ""; dir = filepath.Dir(dir) {
			fmt.Printf("item %v, test dir %v", item, dir)

			allowedID, ok := allowedDevs[dir]
			if !ok {
				continue
			}

			if allowedID != id {
				fmt.Printf("path %q on disallowed device %d", item, id)
				return false
			}

			return true
		}

		panic(fmt.Sprintf("item %v, device id %v not found, allowedDevs: %v", item, id, allowedDevs))
	}

	stat, err := archiver.Scan(target, selectFilter, newScanProgress(gopts))
	if err != nil {
		return err
	}

	arch := archiver.New(repo)
	arch.Excludes = excludes
	arch.SelectFilter = selectFilter

	arch.Warn = func(dir string, fi os.FileInfo, err error) {
		// TODO: make ignoring errors configurable
		fmt.Printf("%s\rwarning for %s: %v\n", ClearLine(), dir, err)
	}

	_, id, err = arch.Snapshot(context.TODO(), newArchiveProgress(gopts, stat), target, opts.Tags, opts.Hostname, parentSnapshotID)
	if err != nil {
		return err
	}

	fmt.Printf("snapshot %s saved\n", id.Str())

	return nil
}

// TODO : maybe check if web client still listening?
func IsProcessBackground() bool {
	return false
}

// TODO: copied from cmd_backup.go
func newScanProgress() *restic.Progress {
	p := restic.NewProgress()

	p.OnUpdate = func(s restic.Stat, d time.Duration, ticker bool) {
		if IsProcessBackground() {
			return
		}

		PrintProgress("[%s] %d directories, %d files, %s", formatDuration(d), s.Dirs, s.Files, formatBytes(s.Bytes))
	}

	p.OnDone = func(s restic.Stat, d time.Duration, ticker bool) {
		PrintProgress("scanned %d directories, %d files in %s\n", s.Dirs, s.Files, formatDuration(d))
	}

	return p
}




// TODO: copied from cmd_backup.go
//
// filterExisting returns a slice of all existing items, or an error if no
// items exist at all.
func filterExisting(items []string) (result []string, err error) {
	for _, item := range items {
		_, err := fs.Lstat(item)
		if err != nil && os.IsNotExist(errors.Cause(err)) {
			continue
		}

		result = append(result, item)
	}

	if len(result) == 0 {
		return nil, errors.Fatal("all target directories/files do not exist")
	}

	return
}

// TODO: copied from cmd_backup.go
//
// gatherDevices returns the set of unique device ids of the files and/or
// directory paths listed in "items".
func gatherDevices(items []string) (deviceMap map[string]uint64, err error) {
	deviceMap = make(map[string]uint64)
	for _, item := range items {
		fi, err := fs.Lstat(item)
		if err != nil {
			return nil, err
		}
		id, err := fs.DeviceID(fi)
		if err != nil {
			return nil, err
		}
		deviceMap[item] = id
	}
	if len(deviceMap) == 0 {
		return nil, errors.New("zero allowed devices")
	}
	return deviceMap, nil
}
