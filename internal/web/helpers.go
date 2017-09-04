package web

import (
	"html/template"
	"fmt"
)

// misc rails-style template helpers

var Helpers = template.FuncMap{
	"HomePath": homePath,
	"RepoPath": repoUrl,
}

func homePath() (string) {
	return "/"
}

func repoUrl(repo Repo) (string) {
	return fmt.Sprintf("/?repo=%s", repo.Name)
}
