package github

import (
	"fmt"
	"testing"
)

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

// TestPullHeadRef checks the refspec that SignOff fetches before it signs. The
// fetch makes the head commit a local object, so `gh signoff` can verify it.
func TestPullHeadRef(t *testing.T) {
	if got := pullHeadRef(123); got != "refs/pull/123/head" {
		t.Errorf("pullHeadRef(123) = %q", got)
	}
}

// TestSignOffAll checks the batch loop. It counts the signed pull requests and
// collects the failures. One failure does not stop the rest.
func TestSignOffAll(t *testing.T) {
	pulls := []PullRequest{{Number: 1}, {Number: 2}, {Number: 3}}
	res := SignOffAll(pulls, func(number int) error {
		if number == 2 {
			return fmt.Errorf("boom")
		}
		return nil
	})
	if res.Signed != 2 {
		t.Errorf("signed = %d, want 2", res.Signed)
	}
	if len(res.Failures) != 1 || res.Failures[0].Number != 2 || res.Failures[0].Error != "boom" {
		t.Errorf("failures = %+v", res.Failures)
	}
}
