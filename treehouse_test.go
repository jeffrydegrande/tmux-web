package main

import (
	"encoding/json"
	"testing"
)

// TestPoolParse checks that the status JSON shape matches the struct. The
// sample is real output from `treehouse status --json`.
func TestPoolParse(t *testing.T) {
	sample := `[{"name":"1","path":"/home/x/.treehouse/voicevo-b667fc/1/voicevo","status":"dirty","lease_id":"","lease_holder":"","leased_at":null,"processes":[]},
	{"name":"2","path":"/home/x/.treehouse/voicevo-b667fc/2/voicevo","status":"leased","lease_id":"abc","lease_holder":"ENG-42","leased_at":"2026-08-15T00:00:00Z","processes":[]}]`

	var pool []worktree
	if err := json.Unmarshal([]byte(sample), &pool); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(pool) != 2 {
		t.Fatalf("got %d worktrees, want 2", len(pool))
	}

	var found string
	for _, w := range pool {
		if w.LeaseHolder == "ENG-42" {
			found = w.Path
		}
	}
	want := "/home/x/.treehouse/voicevo-b667fc/2/voicevo"
	if found != want {
		t.Errorf("reuse path for ENG-42 = %q, want %q", found, want)
	}
}

// TestLeaseParse checks the get --lease --json shape.
func TestLeaseParse(t *testing.T) {
	var l lease
	if err := json.Unmarshal([]byte(`{"path":"/tmp/wt","lease_id":"z"}`), &l); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if l.Path != "/tmp/wt" {
		t.Errorf("path = %q, want /tmp/wt", l.Path)
	}
}
