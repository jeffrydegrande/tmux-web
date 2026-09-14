package treehouse

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// worktree mirrors one entry of `treehouse status --json`.
type worktree struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Status      string `json:"status"`
	LeaseHolder string `json:"lease_holder"`
}

// lease mirrors `treehouse get --lease --json`.
type lease struct {
	Path string `json:"path"`
}

// LeaseWorktree returns a worktree path for holder in the repo at dir. It
// reuses a worktree already leased to holder. Otherwise it leases a new one
// from the pool. treehouse must be initialized in dir.
func LeaseWorktree(dir, holder string) (string, error) {
	if path, err := reuseLease(dir, holder); err != nil {
		return "", err
	} else if path != "" {
		return path, nil
	}

	// treehouse prints banners on stderr, so stdout stays clean for the JSON.
	out, err := runTreehouse(dir, "get", "--lease", "--lease-holder", holder, "--json")
	if err != nil {
		return "", fmt.Errorf("lease worktree: %w", err)
	}
	var l lease
	if err := json.Unmarshal([]byte(out), &l); err != nil {
		return "", fmt.Errorf("parse lease: %w", err)
	}
	if l.Path == "" {
		return "", fmt.Errorf("treehouse returned no path")
	}
	return l.Path, nil
}

func reuseLease(dir, holder string) (string, error) {
	out, err := runTreehouse(dir, "status", "--json")
	if err != nil {
		return "", fmt.Errorf("treehouse status (run 'treehouse init' in %s): %w", dir, err)
	}
	var pool []worktree
	if err := json.Unmarshal([]byte(out), &pool); err != nil {
		return "", fmt.Errorf("parse pool: %w", err)
	}
	for _, w := range pool {
		if w.LeaseHolder == holder {
			return w.Path, nil
		}
	}
	return "", nil
}

func runTreehouse(dir string, args ...string) (string, error) {
	cmd := exec.Command("treehouse", args...)
	cmd.Dir = dir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
