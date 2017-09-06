package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
)

type FormErrors map[string]string

func (a *Repo) Validate() (ok bool, errors FormErrors) {
	errors = make(map[string]string)

	if strings.TrimSpace(a.Name) == "" {
		errors["Name"] = "Please enter a name for the repository."
	}

	re := regexp.MustCompile(string(filepath.Separator) + ".*")
	matched := re.Match([]byte(a.Path))
	if matched == false {
		errors["Path"] = "Please enter a valid path beginning with " + string(filepath.Separator)
	}

	if strings.TrimSpace(a.Password) == "" {
		errors["Password"] = "Please enter a password"
	}

	return len(errors) == 0, errors
}

func fromForm(r *http.Request) Repo {
	return Repo{
		Name:     r.FormValue("name"),
		Path:     r.FormValue("path"),
		Password: r.FormValue("password"),
	}
}

func AddRepoAjaxHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Printf("error parsing form: %s\n", err.Error())
		return
	}

	repo := fromForm(r)
	ok, errors := repo.Validate()

	if !ok {
		fmt.Printf("AddRepoAjaxHandler validation failed: %v\n", errors)

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
	} else {
		fmt.Printf("addRepoHandler validation success\n")

		err := WebConfig.AddRepo(repo)
		// NB: order seems to matter here.
		// 1) content-type
		// 2) cookies
		// 3) status
		// 4) JSON response

		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			fmt.Printf("error adding repo: %s\n", err)
			SaveFlashToCookie(w, "danger_flash", fmt.Sprintf("Error adding new repository: %s", err))
		} else {
			SaveFlashToCookie(w, "success_flash", fmt.Sprintf("New repository \"%s\" added", repo.Name))
		}

		w.WriteHeader(http.StatusOK)

		redirectJs := fmt.Sprintf("{\"on_success\": \"window.location.href='/?repo=%s'\"}", repo.Name)
		w.Write([]byte(redirectJs))
	}
}
