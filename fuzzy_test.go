package main

import (
	"slices"
	"testing"
)

func TestFuzzyFilter(t *testing.T) {
	branches := []string{"main", "feature/login", "fix/logging", "feature/logout", "chore/deps", "bryn/ParseConfig"}

	tests := []struct {
		query string
		want  []string
	}{
		{"", branches},
		{"login", []string{"feature/login", "fix/logging"}},
		{"log", []string{"fix/logging", "feature/login", "feature/logout"}},
		{"fl", []string{"fix/logging", "feature/login", "feature/logout"}},
		{"deps", []string{"chore/deps"}},
		{"LOGIN", nil},
		{"pc", []string{"bryn/ParseConfig"}},
		{"PC", []string{"bryn/ParseConfig"}},
		{"zzz", nil},
	}
	for _, tt := range tests {
		got := fuzzyFilter(tt.query, branches)
		if !slices.Equal(got, tt.want) {
			t.Errorf("fuzzyFilter(%q) = %q, want %q", tt.query, got, tt.want)
		}
	}
}

func TestFuzzyScorePrefersContiguousAndBoundaries(t *testing.T) {
	better, _ := fuzzyScore("log", "feature/login")
	worse, _ := fuzzyScore("log", "feature/large-object-graph")
	if better <= worse {
		t.Errorf("contiguous match scored %d, scattered match scored %d", better, worse)
	}
}

func TestCompletionsBash(t *testing.T) {
	wts := []worktree{
		{path: "/r", branch: "main"},
		{path: "/r/a", branch: "feature/login"},
		{path: "/r/b", branch: "feature/logout"},
		{path: "/r/c", branch: "fix/logging"},
		{path: "/r/d"},
	}

	tests := []struct {
		args []string
		want []string
	}{
		// common prefix "feature/log" still contains "log": readline may insert it
		{[]string{"--bash", "feat"}, []string{"feature/login", "feature/logout"}},
		{[]string{"--bash", "flo"}, []string{"fix/logging", "feature/login", "feature/logout", "flo"}},
		// common prefix "f" would discard the typed "lo": the word is kept as a candidate
		{[]string{"--bash", "lo"}, []string{"fix/logging", "feature/login", "feature/logout", "lo"}},
		{[]string{"--bash", "main"}, []string{"main"}},
		{[]string{"--bash", ""}, []string{"main", "feature/login", "feature/logout", "fix/logging"}},
		{[]string{"lo"}, []string{"fix/logging", "feature/login", "feature/logout"}},
		{nil, []string{"main", "feature/login", "feature/logout", "fix/logging"}},
	}
	for _, tt := range tests {
		got := completions(wts, tt.args)
		if !slices.Equal(got, tt.want) {
			t.Errorf("completions(%q) = %q, want %q", tt.args, got, tt.want)
		}
	}
}

func TestCommonPrefix(t *testing.T) {
	tests := []struct {
		in   []string
		want string
	}{
		{nil, ""},
		{[]string{"abc"}, "abc"},
		{[]string{"feature/login", "feature/logout"}, "feature/log"},
		{[]string{"main", "fix"}, ""},
		{[]string{"héllo", "hé"}, "hé"},
	}
	for _, tt := range tests {
		if got := commonPrefix(tt.in); got != tt.want {
			t.Errorf("commonPrefix(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
