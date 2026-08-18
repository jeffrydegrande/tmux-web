package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

// Window is one tmux window inside a session.
type Window struct {
	Index  string
	Name   string
	Active bool
	Panes  int
}

// Session is one tmux session and its windows.
type Session struct {
	Name     string
	Attached bool
	Windows  []Window
}

// targetPattern limits capture targets to safe characters. It blocks any tmux
// flag or shell metacharacter from reaching the command line.
var targetPattern = regexp.MustCompile(`^[A-Za-z0-9_=./][A-Za-z0-9_:=./-]*$`)

// tmuxField splits tmux -F output on the unit separator. tmux never emits it,
// so window names with spaces stay intact.
const tmuxSep = "\x1f"

// ListSessions returns every running tmux session with its windows. It returns
// an empty slice when the tmux server is not running.
func ListSessions() ([]Session, error) {
	// #{session_attached} is 0 or 1. An empty output means no server.
	out, err := runTmux("list-sessions", "-F",
		"#{session_name}"+tmuxSep+"#{session_attached}")
	if err != nil {
		if noServer(err) {
			return []Session{}, nil
		}
		return nil, err
	}

	var sessions []Session
	for _, line := range splitLines(out) {
		f := strings.Split(line, tmuxSep)
		if len(f) != 2 {
			continue
		}
		s := Session{Name: f[0], Attached: f[1] == "1"}
		windows, err := listWindows(s.Name)
		if err != nil {
			return nil, err
		}
		s.Windows = windows
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func listWindows(session string) ([]Window, error) {
	out, err := runTmux("list-windows", "-t", "="+session, "-F",
		"#{window_index}"+tmuxSep+"#{window_name}"+tmuxSep+
			"#{window_active}"+tmuxSep+"#{window_panes}")
	if err != nil {
		return nil, err
	}

	var windows []Window
	for _, line := range splitLines(out) {
		f := strings.Split(line, tmuxSep)
		if len(f) != 4 {
			continue
		}
		windows = append(windows, Window{
			Index:  f[0],
			Name:   f[1],
			Active: f[2] == "1",
			Panes:  atoi(f[3]),
		})
	}
	return windows, nil
}

// CapturePane returns the visible text of a window's active pane. The target is
// "session:window". It is validated before use.
func CapturePane(target string) (string, error) {
	if !targetPattern.MatchString(target) {
		return "", fmt.Errorf("invalid target")
	}
	// -p prints to stdout. -J joins wrapped lines. "=" forces an exact match so
	// a target can not select the wrong session by prefix.
	out, err := runTmux("capture-pane", "-p", "-J", "-t", "="+target)
	if err != nil {
		return "", err
	}
	return out, nil
}

// EnsureSession starts a detached session in dir when it does not exist.
func EnsureSession(session, dir string) error {
	if _, err := runTmux("has-session", "-t", "="+session); err == nil {
		return nil
	}
	_, err := runTmux("new-session", "-d", "-s", session, "-c", dir)
	return err
}

// NewWindow opens a window that leases-nothing itself; the caller passes the
// leased worktree as dir. It starts Claude Code, then drops into a shell so the
// window (and the lease) stays alive after Claude Code exits.
func NewWindow(session, name, dir, prompt string) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	var claude string
	if prompt == "" {
		claude = "claude"
	} else {
		claude = "claude " + shellQuote(prompt)
	}
	inner := fmt.Sprintf("%s; exec %s", claude, shellQuote(shell))

	_, err := runTmux("new-window", "-t", "="+session, "-n", name, "-c", dir, inner)
	return err
}

// HasWindow reports whether a window with the exact name exists in the session.
func HasWindow(session, name string) (bool, error) {
	out, err := runTmux("list-windows", "-t", "="+session, "-F", "#{window_name}")
	if err != nil {
		if noServer(err) {
			return false, nil
		}
		return false, err
	}
	return slices.Contains(splitLines(out), name), nil
}

func runTmux(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux %s: %w: %s",
			strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func noServer(err error) bool {
	return strings.Contains(err.Error(), "no server running") ||
		strings.Contains(err.Error(), "no current session")
}

func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return n
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// shellQuote wraps a string in single quotes for safe use in a sh command. It
// escapes embedded single quotes with the '\” idiom.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
