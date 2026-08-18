# AGENTS.md

Guidance for agents that work in this repo.

## What this is

A single Go binary. It serves a mobile web UI over HTTP. It lists tmux sessions
and starts new tmux windows that run Claude Code in a treehouse worktree. See
`README.md` for the user view.

## Layout

- `main.go`: HTTP server, routes, and the new-window flow.
- `config.go`, `default_config.go`: TOML config and defaults.
- `tmux.go`: tmux commands (list, capture, new session, new window).
- `treehouse.go`: worktree leases.
- `linear.go`: Linear API client for assigned tickets.
- `github.go`: `gh pr list` wrapper for open pull requests.
- `templates/*.html`: HTML templates. HTMX drives the interactions.
- `static/`: CSS and the embedded HTMX script.
- `systemd/`: the user service unit.

Templates, CSS, and HTMX are embedded with `//go:embed`. The binary is
self-contained.

## Build, test, format

```bash
go build ./...
go test ./...
gofmt -l .
```

Run all three before you commit. `gofmt -l .` must print nothing.

## Conventions

- External tools run through small wrappers: `runTmux`, `runTreehouse`, and the
  `exec.Command` calls in `github.go`. Add new tool calls the same way.
- Keep parsing logic in pure functions so tests can cover it. See `parsePulls`
  and `keepIssue`. Tests check the JSON shape against real command output.
- Validate any value that reaches a command line. See `targetPattern` and
  `sanitizeName`.
- Match the surrounding style. Short doc comments on exported names.

## Writing style

Write in Simplified Technical English. Short sentences. One idea per sentence.
Active voice. Common words. Use a plain dash, never an em dash. This covers
code comments, docs, and commit messages.
