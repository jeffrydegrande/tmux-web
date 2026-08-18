package main

import "testing"

// TestParsePulls checks that the gh pr list JSON shape maps to pull requests.
// The sample is real output from `gh pr list --json number,title,author,headRefName`.
func TestParsePulls(t *testing.T) {
	sample := `[
	  {"number":42,"title":"Add cache layer","author":{"login":"alice"},"headRefName":"feat/cache"},
	  {"number":7,"title":"Fix flaky test","author":{"login":"bob"},"headRefName":"fix/flaky"}
	]`

	pulls, err := parsePulls([]byte(sample))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(pulls) != 2 {
		t.Fatalf("got %d pulls, want 2", len(pulls))
	}
	if pulls[0].Number != 42 || pulls[0].Title != "Add cache layer" ||
		pulls[0].Author != "alice" || pulls[0].Branch != "feat/cache" {
		t.Errorf("first pull = %+v", pulls[0])
	}
}

// TestParsePullsEmpty checks that no open pull requests parses to an empty list.
func TestParsePullsEmpty(t *testing.T) {
	pulls, err := parsePulls([]byte(`[]`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(pulls) != 0 {
		t.Errorf("got %d pulls, want 0", len(pulls))
	}
}
