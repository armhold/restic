package web

import (
	"fmt"
	"html/template"
	"net/http"
	"time"
)

// misc rails-style template helpers

var Helpers = template.FuncMap{
	"HomePath":   homePath,
	"FormatTime": FormatTime,
}

func homePath() string {
	return "/"
}

type Navigation struct {
	req *http.Request
	Tab string
}

func SnapshotsUrl() string {
	return "/snapshots"
}

// return url to snapshots tab
func (n *Navigation) SnapshotsUrl() string {
	return SnapshotsUrl()
}

// return url to paths tab
func (n *Navigation) PathsUrl() string {
	return "/paths"
}

func (n *Navigation) ExcludesUrl() string {
	return "/excludes"
}

func (n *Navigation) ScheduleUrl() string {
	return "/schedule"
}

func (n *Navigation) BackupUrl() string {
	return "/backup"
}

func (n *Navigation) BrowseUrl() string {
	return "/browse"
}

func (n *Navigation) SnapshotUrl(repo, snapshotId string) string {
	s := fmt.Sprintf("/snaps/%s", snapshotId)
	//u, err := url.Parse(s)
	//if err != nil {
	//	// TODO: better way to handle errors in a helper func
	//	fmt.Println(err)
	//	return ""
	//}
	//
	//q := u.Query()
	//q.Set("snapshotId", snapshotId)
	//u.RawQuery = q.Encode()
	//return u.String()
	return s
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
