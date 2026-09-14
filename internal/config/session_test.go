package config

import "testing"

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
