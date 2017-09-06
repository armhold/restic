package web

import (
	"html/template"
	"fmt"
	"net/http"
	"net/url"
	"log"
	"time"
)

// misc rails-style template helpers

var Helpers = template.FuncMap{
	"HomePath":     homePath,
	"RepoPath":     repoUrl,
	"SnapshotTime": SnapshotTime,
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

// return url to snapshots tab, while preserving current repo id
func (n *Navigation) SnapshotsUrl() (string) {
	return "/snapshots?repo=" + n.req.FormValue("repo")
}

// return url to paths tab, while preserving current repo id
func (n *Navigation) PathsUrl() (string) {
	return "/paths?repo=" + n.req.FormValue("repo")
}

// return url to excludes tab, while preserving current repo id
func (n *Navigation) ExcludesUrl() (string) {
	return "/excludes?repo=" + n.req.FormValue("repo")
}

// return url to schedule tab, while preserving current repo id
func (n *Navigation) ScheduleUrl() (string) {
	return "/schedule?repo=" + n.req.FormValue("repo")
}

func (n *Navigation) CssForTab(tab string) (string) {
	if n.req.URL.Path[1:] == tab {
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

func SnapshotTime(t time.Time) (string) {
	return t.Format(TimeFormat)
}

// TODO: copied from main/table.go
const TimeFormat = "2006-01-02 15:04:05"
