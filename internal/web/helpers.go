package web

import (
	"html/template"
	"fmt"
	"net/http"
	"net/url"
	"log"
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

type Navigation struct {
	req *http.Request
}

func (n *Navigation) Snapshots() (bool) {
	return n.req.FormValue("tab") == "snaps"
}

func (n *Navigation) Paths() (bool) {
	return n.req.FormValue("tab") == "paths"
}

func (n *Navigation) Exclude() (bool) {
	return n.req.FormValue("tab") == "exclude"
}

func (n *Navigation) Foo() (bool) {
	return n.req.FormValue("tab") == "foo"
}

func (n *Navigation) CssForTab(tab string) (string) {
	if n.req.FormValue("tab") == tab {
		return "active"
	}

	return ""
}

func (n *Navigation) HrefForTab(tab string) (string) {
	orig := n.req.URL.String()

	u, err := url.Parse(orig)
	if err != nil {
		log.Printf("error parsing url: %s", err)
		return ""
	}
	q := u.Query()
	q.Set("tab", tab)
	u.RawQuery = q.Encode()

	return u.String()
}
