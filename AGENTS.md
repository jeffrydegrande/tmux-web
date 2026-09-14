# AGENTS.md

Guidance for agents that work in this repo.

## What this is

A Go module with one binary. The binary serves a mobile web UI over HTTP. It
lists tmux sessions and starts new tmux windows that run Claude Code in a
treehouse worktree. See `README.md` for the user view. The module is
`github.com/jeffrydegrande/tmux-web`.

## Layout

- `cmd/web/main.go`: HTTP server, routes, and the new-window flow.
- `internal/config/`: TOML config, defaults, and the project search-dir scan.
- `internal/tmux/`: tmux commands (list, capture, new session, new window).
- `internal/treehouse/`: worktree leases.
- `internal/linear/`: Linear API client for assigned tickets.
- `internal/github/`: `gh pr list` wrapper for open pull requests.
- `web/templates/*.html`: HTML templates. HTMX drives the interactions.
- `web/static/`: CSS and the embedded HTMX script.
- `web/web.go`: embeds the templates and static files.
- `systemd/`: the user service unit.

The `web` package embeds the templates and static files with `//go:embed`. A
Go embed pattern can not reach a parent directory, so the embed lives in `web`,
next to the files, not in `cmd/web`. The binary is self-contained.

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

## Maintaining this file

Keep this file for knowledge useful to almost every future agent session in this project.
Do not repeat what the codebase already shows; point to the authoritative file or command instead.
Prefer rewriting or pruning existing entries over appending new ones.
When updating this file, preserve this bar for all agents and keep entries concise.
