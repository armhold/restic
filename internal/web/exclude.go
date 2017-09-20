package web

import (
	"fmt"
	"net/http"
	"sort"
)

func excludeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("excludeHandler\n")

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
		Excludes     []string
		Tab          string
	}{
		Repos:        WebConfig.Repos,
		CurrRepoName: currRepoName,
		Flash:        flash,
		Css_class:    cssClassForRepo,
		Nav:          &Navigation{req: r, Tab: "excludes"},
		Excludes:     sortedExcludes(repo),
	}

	if err := templates.ExecuteTemplate(w, "excludes.html", data); err != nil {
		fmt.Printf("%s\n", err.Error())
	}

	fmt.Printf("sucessful exit excludesHandler()\n")
}

// add/remove exclusion to/from the backup list
// when method=DELETE, the exclusion is deleted
func addDeleteExcludeAjaxHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Printf("error parsing form: %s\n", err.Error())
		return
	}

	exclude := r.FormValue("exclude")
	createExclude := r.Method != "DELETE"
	repoName := r.FormValue("repo")

	fmt.Printf("update repo: \"%s\"\n", repoName)
	fmt.Printf("\tmethod: \"%s\"\n", r.Method)
	fmt.Printf("\texclude: \"%s\"\n", exclude)
	fmt.Printf("\tcreateExclude: %t\n", createExclude)

	currRepoName := r.FormValue("repo")
	repo, ok := findCurrRepoByName(currRepoName, WebConfig.Repos)

	if !ok {
		sendErrorToJs(w, fmt.Sprintf("could not find repo: %s", repoName))
		return
	}

	// TODO: mutex on WebConfig before we modify it
	if createExclude {
		repo.BackupPaths.Excludes[exclude] = true
	} else {
		fmt.Printf("deleting key: \"%s\"\n", exclude)
		delete(repo.BackupPaths.Excludes, exclude)
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
		SaveFlashToCookie(w, "danger_flash", fmt.Sprintf("Error adding new exclude: %s", err))
	}

	w.WriteHeader(http.StatusOK)
	executeJs := fmt.Sprintf("{\"on_success\": \"console.log('ok');\"}")
	w.Write([]byte(executeJs))
}

func sortedExcludes(repo *Repo) []string {
	result := repo.BackupPaths.GetExcludes()

	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	return result
}
