package main

import (
	"testing"
)

func TestParseCmd(t *testing.T) {
	var tests = []struct {
		input        string
		expectedCmd  string
		expectedArgs string
		expectedErr  string
	}{
		{"", "", "", ""},
		{"ls", "ls", "", ""},
		{"lsx", "", "", "no such command: lsx"},
		{"lsx ", "", "", "no such command: lsx"},
		{"ls x", "ls", "x", ""},
		{"done", "done", "", ""},
		{"done x", "", "", "no such command: done x"},
	}

	for _, test := range tests {
		gotCmd, gotArgs, gotErr := parseCmd(test.input)

		if errString(gotErr) != test.expectedErr {
			t.Errorf("expected err: \"%v\", got: \"%v\", for input %s", test.expectedErr, errString(gotErr), test.input)
		}

		if gotCmd != test.expectedCmd {
			t.Errorf("expected cmd: %s, got: %s, for input %s", test.expectedCmd, gotCmd, test.input)
		}

		if gotArgs != test.expectedArgs {
			t.Errorf("expected args: %s, got: %s, for input %s", test.expectedArgs, gotArgs, test.input)
		}
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}

func TestLongestCommonPrefix(t *testing.T) {
	var tests = []struct {
		input    []string
		expected string
	}{
		{[]string{"", "", ""}, ""},
		{[]string{"a", "a", "a"}, "a"},
		{[]string{"a", "ab", "ab"}, "a"},
		{[]string{"abc", "abc", "cde", ""}, ""},
		{[]string{"aaa", "aab", ""}, ""},
		{[]string{"aaa", "aab", "aac"}, "aa"},
		{[]string{"bbb", "aab", "aac"}, ""},
		{[]string{"aaaa", "aaaa", "aaaa"}, "aaaa"},
		{[]string{"aaaaa", "aaaab", "aaaac"}, "aaaa"},
	}

	for _, test := range tests {
		got := longestCommonPrefix(test.input)

		if got != test.expected {
			t.Errorf("expected: %s, got: %s, for input %s", test.expected, got, test.input)
		}
	}
}
