package web

import (
	"fmt"
	"github.com/restic/restic/internal/restic"
	"net/http"
	"strings"
	"context"
)

func snapshotsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("snapshotsHandler\n")
	fmt.Printf("path: %q\n", r.URL.Path)

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
	repo, ok := findCurrRepoByName(currRepoName, WebConfig.Repos)
	if ok {
		snaps, err = listSnapshots(repo)
		if err != nil {
			fmt.Printf("listSnapshots: %s\n", err.Error())

			// NB: don't call SaveFlashToCookie() because we want it to render immediately here, not after redirect
			flash.Danger += fmt.Sprintf("error listing snapshots: %s", err)
		}
	}

	data := struct {
		Repos        []*Repo
		CurrRepoName string
		Flash        Flash
		Css_class    func(repoName string) string
		Snapshots    restic.Snapshots
		Nav          *Navigation
	}{
		Repos:     WebConfig.Repos,
		CurrRepoName: currRepoName,
		Flash:     flash,
		Css_class: cssClassForRepo,
		Snapshots: snaps,
		Nav:       &Navigation{req: r, Tab: "snapshots"},
	}

	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit snapshotsHandler()\n")
}

type deleteSnapshot struct {
	repo string
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

func DeleteSnapshotAjaxHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("DeleteSnapshotAjaxHandler\n")

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

	currRepo, ok := findCurrRepoByName(d.repo, WebConfig.Repos)

	if ! ok {
		sendErrorToJs(w, fmt.Sprintf("could not find repo: %s", d.repo))
		return
	}

	err = removeSnapshot(currRepo, d.snapshotId)

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

func removeSnapshot(repo *Repo, snapId string) (error) {
	r, err := OpenRepository(repo.Path, repo.Password)
	if err != nil {
		return err
	}

	lock, err := lockRepoExclusive(r)
	defer unlockRepo(lock)
	if err != nil {
		return err
	}

	id, err := restic.FindSnapshot(r, snapId)
	if err != nil {
		return err
	}

	h := restic.Handle{Type: restic.SnapshotFile, Name: id.String()}
	err = r.Backend().Remove(context.TODO(), h)
	if err != nil {
		return err
	}

	return nil
}
