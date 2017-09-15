package web

import (
	"html/template"
	"net/http"
	"net/url"
	"time"
	"fmt"
)

// misc rails-style template helpers

var Helpers = template.FuncMap{
	"HomePath":     homePath,
	"RepoPath":     repoUrl,
	"FormatTime":   FormatTime,
}

func homePath() string {
	return "/"
}

func repoUrl(repo Repo) string {
	return SnapshotsUrl(repo.Name)
}

type Navigation struct {
	req *http.Request
	Tab string
}

func SnapshotsUrl(repo string) string {
	return "/snapshots?repo=" + repo
}

// return url to snapshots tab, while preserving current repo id
func (n *Navigation) SnapshotsUrl() string {
	return SnapshotsUrl(n.req.FormValue("repo"))
}

// return url to paths tab, while preserving current repo id
func (n *Navigation) PathsUrl() string {
	return "/paths?repo=" + n.req.FormValue("repo")
}

// return url to excludes tab, while preserving current repo id
func (n *Navigation) ExcludesUrl() string {
	return "/excludes?repo=" + n.req.FormValue("repo")
}

// return url to schedule tab, while preserving current repo id
func (n *Navigation) ScheduleUrl() string {
	return "/schedule?repo=" + n.req.FormValue("repo")
}

func (n *Navigation) BackupUrl() string {
	return "/backup?repo=" + n.req.FormValue("repo")
}

func (n *Navigation) BrowseUrl() string {
	return "/browse?repo=" + n.req.FormValue("repo")
}

func (n *Navigation) RestoreUrl(repo, snapshotId string) string {
	restoreUrl, err := url.Parse("/nav")
	if err != nil {
		// TODO: better way to handle errors in a helper func
		fmt.Println(err)
		return ""
	}

	q := restoreUrl.Query()
	q.Set("repo", repo)
	q.Set("snapshotId", snapshotId)
	restoreUrl.RawQuery = q.Encode()
	return restoreUrl.String()
}



func (n *Navigation) CssForTab(tab string) string {
	if n.req.URL.Path[1:] == tab {
		return "active"
	}

	return ""
}

func FormatTime(t time.Time) string {
	return t.Format(TimeFormat)
}

// TODO: copied from main/table.go
const TimeFormat = "2006-01-02 15:04:05"
