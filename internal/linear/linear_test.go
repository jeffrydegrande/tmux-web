package linear

import "testing"

func TestKeepIssue(t *testing.T) {
	cases := []struct {
		stateType string
		stateName string
		want      bool
	}{
		{"started", "In Progress", true},
		{"unstarted", "Todo", true},
		{"backlog", "Backlog", true},
		{"completed", "Done", false},
		{"canceled", "Canceled", false},
		{"started", "In Review", false},
		{"started", "Code Review", false},
		{"started", "review", false},
	}
	for _, c := range cases {
		if got := keepIssue(c.stateType, c.stateName); got != c.want {
			t.Errorf("keepIssue(%q, %q) = %v, want %v", c.stateType, c.stateName, got, c.want)
		}
	}
}
