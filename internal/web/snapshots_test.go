package web

import (
	"testing"
	"reflect"
)

func TestSplitSubdirs(t *testing.T) {
	var tests = []struct {
		path     string
		expected []string
	}{
		{"/foo/bar/baz", []string{"foo", "bar", "baz"}},
		{"foo/bar/baz", []string{"foo", "bar", "baz"}},
		{"/foo", []string{"foo"}},
		{"/foo/", []string{"foo"}},
		{"foo/", []string{"foo"}},
		{"foo", []string{"foo"}},
	}

	for _, test := range tests {
		got := splitIntoDirs(test.path)

		if !reflect.DeepEqual(got, test.expected) {
			t.Errorf("expected: \"%#v\", got: \"%#v\", for input %s", test.expected, got, test.path)
		}
	}
}
