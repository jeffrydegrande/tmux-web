package main

import "testing"

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
