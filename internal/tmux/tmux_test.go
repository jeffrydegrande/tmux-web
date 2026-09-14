package tmux

import (
	"errors"
	"testing"
)

func TestNoServer(t *testing.T) {
	// Fresh boot: the socket file does not exist yet.
	fresh := errors.New("tmux list-sessions: exit status 1: error connecting to /tmp/tmux-1000/default (No such file or directory)")
	if !noServer(fresh) {
		t.Error("fresh-boot connect error must count as no server")
	}
	// Other known no-server messages.
	for _, msg := range []string{"no server running on /tmp/x", "no current session"} {
		if !noServer(errors.New(msg)) {
			t.Errorf("%q must count as no server", msg)
		}
	}
	// A real error must not be swallowed.
	if noServer(errors.New("tmux new-window: exit status 1: can't find pane")) {
		t.Error("a real tmux error must not count as no server")
	}
}

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
