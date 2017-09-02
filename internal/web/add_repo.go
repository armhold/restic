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

func AddRepoHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("addRepoHandler\n")
	w.Header().Set("Content-Type", "application/json")


	err := r.ParseForm()
	if err != nil {
		fmt.Printf("error parsing form: %s\n", err.Error())
		return
	}
	fmt.Printf("addRepoHandler %v\n", r.Form)

	addRepo := &AddRepo{
		Name: r.FormValue("name"),
		Path: r.FormValue("path"),
		Password: r.FormValue("password"),
	}

	if ! addRepo.Validate() {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("addRepoHandler validation failed: %v\n", addRepo.Errors)
	} else {
		w.WriteHeader(http.StatusOK)
		//http.Redirect(w, r, "/", http.StatusSeeOther)
		fmt.Printf("addRepoHandler validation success\n")
	}

	w.Header().Set("Content-Type", "application/json")


	if err := json.NewEncoder(w).Encode(addRepo.Errors); err != nil {
		fmt.Printf("error encoding response %s\n", err)
		return
	}

	fmt.Printf("addRepoHandler success\n")
}

