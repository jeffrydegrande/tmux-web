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

// TestParseHeadRefOid checks that the gh pr view JSON maps to the head sha.
// The sample is real output from `gh pr view <n> --json headRefOid`.
func TestParseHeadRefOid(t *testing.T) {
	sample := `{"headRefOid":"9fceb02d0ae598e95dc970b74767f19372d61af8"}`
	sha, err := parseHeadRefOid([]byte(sample))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if sha != "9fceb02d0ae598e95dc970b74767f19372d61af8" {
		t.Errorf("sha = %q", sha)
	}
}

// TestParseHeadRefOidEmpty checks that a missing sha is an error.
func TestParseHeadRefOidEmpty(t *testing.T) {
	if _, err := parseHeadRefOid([]byte(`{"headRefOid":""}`)); err == nil {
		t.Error("want error for empty sha")
	}
}
