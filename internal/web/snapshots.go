package web

import (
	"context"
	"fmt"
	"github.com/restic/restic/internal/repository"
	"github.com/restic/restic/internal/restic"
	"net/http"
	"sort"
	"strings"
)

func listSnapshotsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("listSnapshotsHandler\n")
	fmt.Printf("path: %q\n", r.URL.Path)

	repo := getRepo()
	defer releaseRepo()

	//session, _ := sessionManager.GetOrCreateSession(w, r)
	//session.Set("current time", time.Now())

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

	var snaps restic.Snapshots
	snaps, err = listSnapshots(repo)
	if err != nil {
		fmt.Printf("listSnapshots: %s\n", err.Error())

		// NB: don't call SaveFlashToCookie() because we want it to render immediately here, not after redirect
		flash.Danger += fmt.Sprintf("error listing snapshots: %s", err)
	}

	data := struct {
		CurrRepoName string
		Flash        Flash
		Css_class    func(repoName string) string
		Snapshots    restic.Snapshots
		Nav          *Navigation
		SnapSelected bool
	}{
		CurrRepoName: currRepoName,
		Flash:        flash,
		Css_class:    cssClassForRepo,
		Snapshots:    snaps,
		Nav:          &Navigation{req: r, Tab: "snapshots"},
		SnapSelected: false,
	}

	if err := templates.ExecuteTemplate(w, "snapshots.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("successful exit listSnapshotsHandler()\n")
}

type deleteSnapshot struct {
	repo       string
	snapshotId string
}

func fromForm(r *http.Request) deleteSnapshot {
	result := deleteSnapshot{}
	result.repo = r.FormValue("repo")
	result.snapshotId = r.FormValue("snapshotId")

	fmt.Printf("deleteSnap: %#v\n", result)

	return result
}

func (d *deleteSnapshot) Validate() (ok bool, errors FormErrors) {
	errors = make(map[string]string)

	if strings.TrimSpace(d.repo) == "" {
		errors["repo"] = "repository name missing"
	}

	if strings.TrimSpace(d.snapshotId) == "" {
		errors["snapshotId"] = "snapshot ID missing"
	}

	return len(errors) == 0, errors
}

func deleteSnapshotAjaxHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("deleteSnapshotAjaxHandler\n")

	repo := getRepo()
	defer releaseRepo()

	err := r.ParseForm()
	if err != nil {
		fmt.Printf("error parsing form: %s\n", err.Error())
		return
	}

	fmt.Printf("received form: %#v\n", r.Form)

	d := fromForm(r)
	ok, errors := d.Validate()
	if !ok {
		sendErrorMapToJs(w, errors)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if !ok {
		sendErrorToJs(w, fmt.Sprintf("could not find repo: %s", d.repo))
		return
	}

	err = removeSnapshot(repo, d.snapshotId)

	if err != nil {
		msg := fmt.Sprintf("Error deleting snapshot: %s", err)
		fmt.Println(msg)
		SaveFlashToCookie(w, "danger_flash", msg)
	} else {
		SaveFlashToCookie(w, "success_flash", fmt.Sprintf("Snapshot \"%s\" deleted", d.snapshotId))
	}

	w.WriteHeader(http.StatusOK)

	redirectJs := fmt.Sprintf("{\"on_success\": \"window.location.href='/snapshots?repo=%s'\"}", d.repo)
	w.Write([]byte(redirectJs))
}

func removeSnapshot(repo *repository.Repository, snapId string) error {
	lock, err := lockRepoExclusive(repo)
	defer unlockRepo(lock)
	if err != nil {
		return err
	}

	id, err := restic.FindSnapshot(repo, snapId)
	if err != nil {
		return err
	}

	h := restic.Handle{Type: restic.SnapshotFile, Name: id.String()}
	return repo.Backend().Remove(context.TODO(), h)
}

func listSnapshots(repo restic.Repository) (restic.Snapshots, error) {
	var snaps restic.Snapshots

	snaps = restic.FindFilteredSnapshots(context.TODO(), repo, "", []restic.TagList{}, []string{})
	sort.Sort(snaps)

	return snaps, nil
}
