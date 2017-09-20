package web

import (
	"context"
	"fmt"
	"github.com/restic/restic/internal/archiver"
	"github.com/restic/restic/internal/errors"
	"github.com/restic/restic/internal/filter"
	"github.com/restic/restic/internal/fs"
	"github.com/restic/restic/internal/restic"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
	if !ok {
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

	if err := templates.ExecuteTemplate(w, "backup.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit backupHandler()\n")
}

func runBackupAjaxHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("runBackupAjaxHandler\n")

	err := r.ParseForm()
	if err != nil {
		fmt.Printf("error parsing form: %s\n", err.Error())
		return
	}

	currRepoName := r.FormValue("repo")
	repo, ok := findCurrRepoByName(currRepoName, WebConfig.Repos)

	w.Header().Set("Content-Type", "application/json")

	if !ok {
		sendErrorToJs(w, fmt.Sprintf("could not find repo: %s", currRepoName))
		return
	}

	go func() {
		err := runBackup(repo)

		bs := BackupStatus{RepoName: currRepoName, PercentDone: 100}

		if err != nil {
			bs.Error = fmt.Sprintf("%s: backup failed: %s", currRepoName, err.Error())
		} else {
			bs.StatusMsg = fmt.Sprintf("%s: backup complete", currRepoName)
		}

		UpdateStatusBlocking(bs)
	}()

	w.WriteHeader(http.StatusOK)
	executeJs := fmt.Sprintf("{\"on_success\": \"backup started for %s\"}", currRepoName)
	w.Write([]byte(executeJs))
}

func runBackup(r *Repo) error {
	target := r.BackupPaths.GetPaths()

	target, err := filterExisting(target)
	if err != nil {
		return err
	}

	// allowed devices
	// TODO: maybe make this an option in repo config for web ui
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
	//err = errors.New("fake an error")
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
	hostname := ""
	id, err := restic.FindLatestSnapshot(context.TODO(), repo, target, []restic.TagList{}, hostname)
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

	stat, err := archiver.Scan(target, selectFilter, newScanProgress(r.Name))
	if err != nil {
		return err
	}

	arch := archiver.New(repo)
	arch.Excludes = excludes
	arch.SelectFilter = selectFilter

	arch.Warn = func(dir string, fi os.FileInfo, err error) {
		// TODO: make ignoring errors configurable
		clearLine := "\n"
		fmt.Printf("%s\rwarning for %s: %v\n", clearLine, dir, err)
	}

	var tags []string
	timeStamp := time.Now()
	_, id, err = arch.Snapshot(context.TODO(), newArchiveProgress(r.Name, false, stat), target, tags, hostname, parentSnapshotID, timeStamp)
	if err != nil {
		return err
	}
	fmt.Printf("snapshot %s saved\n", id.Str())

	return nil
}

// TODO: copied from cmd_backup.go
func newArchiveProgress(repoName string, quiet bool, todo restic.Stat) *restic.Progress {
	if quiet {
		return nil
	}

	archiveProgress := restic.NewProgress()

	var bps, eta uint64
	itemsTodo := todo.Files + todo.Dirs

	archiveProgress.OnUpdate = func(s restic.Stat, d time.Duration, ticker bool) {
		if IsProcessBackground() {
			return
		}

		sec := uint64(d / time.Second)
		if todo.Bytes > 0 && sec > 0 && ticker {
			bps = s.Bytes / sec
			if s.Bytes >= todo.Bytes {
				eta = 0
			} else if bps > 0 {
				eta = (todo.Bytes - s.Bytes) / bps
			}
		}

		itemsDone := s.Files + s.Dirs

		status1 := fmt.Sprintf("[%s] %s  %s/s  %s / %s  %d / %d items  %d errors  ",
			formatDuration(d),
			formatPercent(s.Bytes, todo.Bytes),
			formatBytes(bps),
			formatBytes(s.Bytes), formatBytes(todo.Bytes),
			itemsDone, itemsTodo,
			s.Errors)
		status2 := fmt.Sprintf("ETA %s ", formatSeconds(eta))

		stdoutTerminalWidth := 80

		if w := stdoutTerminalWidth; w > 0 {
			maxlen := w - len(status2) - 1

			if maxlen < 4 {
				status1 = ""
			} else if len(status1) > maxlen {
				status1 = status1[:maxlen-4]
				status1 += "... "
			}
		}

		PrintProgress("%s%s", status1, status2)

		percent := int(100.0 * float64(s.Bytes) / float64(todo.Bytes))
		if percent > 100 {
			percent = 100
		}

		// TODO: don't seem to get called often when run under "fresh", maybe because it's no longer
		// running connected to a terminal.
		bs := BackupStatus{RepoName: repoName, PercentDone: percent, StatusMsg: "", Indeterminate: false}
		UpdateStatus(bs)
		fmt.Printf("updated: %v\n", bs)
	}

	archiveProgress.OnDone = func(s restic.Stat, d time.Duration, ticker bool) {
		fmt.Printf("\nduration: %s, %s\n", formatDuration(d), formatRate(todo.Bytes, d))
	}

	return archiveProgress
}

// TODO : maybe check if web client still listening?
func IsProcessBackground() bool {
	return false
}

func newScanProgress(repoName string) *restic.Progress {
	p := restic.NewProgress()

	p.OnStart = func() {
		bs := BackupStatus{RepoName: repoName, PercentDone: 100, StatusMsg: "Scanning", Indeterminate: true}
		UpdateStatus(bs)
	}

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

// TODO: coped from global.go
// PrintProgress wraps fmt.Printf to handle the difference in writing progress
// information to terminals and non-terminal stdout
func PrintProgress(format string, args ...interface{}) {
	var (
		message         string
		carriageControl string
	)
	message = fmt.Sprintf(format, args...)

	if !(strings.HasSuffix(message, "\r") || strings.HasSuffix(message, "\n")) {
		carriageControl = "\n"
		message = fmt.Sprintf("%s%s", message, carriageControl)
	}

	fmt.Print(message)
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
