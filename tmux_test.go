package main

import "testing"

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"hello":           `'hello'`,
		"work on ENG-1":   `'work on ENG-1'`,
		"it's a trap":     `'it'\''s a trap'`,
		"$(rm -rf /); ls": `'$(rm -rf /); ls'`,
		"back`tick`":      "'back`tick`'",
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"ENG-123":      "ENG-123",
		"ABC-89":       "ABC-89",
		"feat.thing":   "feat_thing",
		"a:b":          "a_b",
		"drop; rm -rf": "drop__rm_-rf",
		"spaces here":  "spaces_here",
	}
	for in, want := range cases {
		if got := sanitizeName(in); got != want {
			t.Errorf("sanitizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRepoSession(t *testing.T) {
	cases := map[string]string{
		"/home/x/Code/voicevo": "voicevo",
		"/home/x/my.app":       "my_app",
		"/home/x/a:b":          "a_b",
	}
	for path, want := range cases {
		r := Repo{Path: path}
		if got := r.Session(); got != want {
			t.Errorf("Repo{%q}.Session() = %q, want %q", path, got, want)
		}
	}
}

func TestTargetPattern(t *testing.T) {
	ok := []string{"my-app:1", "voicevo:12", "a_b:0"}
	bad := []string{"my-app:1;rm", "$(x)", "a b:1", "-X", "a|b"}
	for _, s := range ok {
		if !targetPattern.MatchString(s) {
			t.Errorf("targetPattern rejected valid target %q", s)
		}
	}
	for _, s := range bad {
		if targetPattern.MatchString(s) {
			t.Errorf("targetPattern accepted unsafe target %q", s)
		}
	}
}
