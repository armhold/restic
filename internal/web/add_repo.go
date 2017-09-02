package web

import (
	"regexp"
	"path/filepath"
	"strings"
	"net/http"
	"encoding/json"
	"fmt"
)

type AddRepo struct {
	Name    string  `json:"name"`
	Path    string  `json:"path"`
	Password string `json:"password"`
	Errors  map[string]string `json:"errors"`
}

func (a *AddRepo) Validate() bool {
	a.Errors = make(map[string]string)

	if strings.TrimSpace(a.Name) == "" {
		a.Errors["Name"] = "Please enter a name for the repository."
	}

	re := regexp.MustCompile(string(filepath.Separator) + ".*")
	matched := re.Match([]byte(a.Path))
	if matched == false {
		a.Errors["Path"] = "Please enter a valid path beginning with " + string(filepath.Separator)
	}

	if strings.TrimSpace(a.Password) == "" {
		a.Errors["Password"] = "Please enter a password"
	}

	return len(a.Errors) == 0
}

func AddRepoAjaxHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("addRepoHandler\n")

	err := r.ParseForm()
	if err != nil {
		fmt.Printf("error parsing form: %s\n", err.Error())
		return
	}
	fmt.Printf("AddRepoAjaxHandler received form data: %v\n", r.Form)

	addRepo := &AddRepo{
		Name: r.FormValue("name"),
		Path: r.FormValue("path"),
		Password: r.FormValue("password"),
	}

	if ! addRepo.Validate() {
		fmt.Printf("AddRepoAjaxHandler validation failed: %v\n", addRepo.Errors)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		if err := json.NewEncoder(w).Encode(addRepo.Errors); err != nil {
			fmt.Printf("error encoding response %s\n", err)
			return
		}
	} else {
		//w.WriteHeader(http.StatusOK)
		fmt.Printf("addRepoHandler validation success\n")
		SaveFlashToCookie(w, "success_flash", []byte(fmt.Sprintf("New repository \"%s\" added", addRepo.Name)))
		//w.Write([]byte("{'on_success': 'window.location.href = \"/?foo=bar\"'}"))

		// add_repo.html is hard-coded to load / upon success
		//http.Redirect(w, r, "/", http.StatusSeeOther)
	}

	fmt.Printf("returning from AddRepoAjaxHandler\n")
}

