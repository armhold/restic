package web

import (
	"net/http"
	"testing"
)

// since we parse restful urls by hand. If we move to gorilla.mux or similar we can ditch this.
func TestGetSnapshotId(t *testing.T) {
	var tests = []struct {
		url    string
		snapId string
	}{
		{"/0123abcd", "0123abcd"},
		{"/0123abcd/", "0123abcd"},
		{"/0123abcd/foo", "0123abcd"},
		{"/0123abcd/foo/bar/baz", "0123abcd"},
	}

	for _, tt := range tests {
		req, err := http.NewRequest("GET", tt.url, nil)
		if err != nil {
			t.Fatal(err)
		}

		actual := getSnapshotId(req)
		if actual != tt.snapId {
			t.Fatalf("for: %s expected: %s, got: %s", tt.url, tt.snapId, actual)
		}

	}
}
