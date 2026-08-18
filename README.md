# tmux-web

A small Go server. It shows your tmux sessions in a mobile web UI. You start
new work from your phone. It runs Claude Code in a fresh tmux window, in a
[treehouse](https://github.com/jeffrydegrande/treehouse) worktree.

The UI is plain HTML with [HTMX](https://htmx.org). Everything is embedded in
one binary. The page needs no internet.

## What it does

- Lists every tmux session and window.
- Shows a text snapshot of any window.
- Starts a new window for a repo you pick. A prompt is optional. The picker
  also lists projects found under your `search_dirs`.
- Lists your assigned Linear tickets, grouped per project. One tap starts a
  window for a ticket. Tickets in review do not show; the pull requests list
  covers those.
- Lists open GitHub pull requests, grouped per project. Tap a pull request,
  then pick a prompt: `deep review on PR <number>` or
  `fix merge conflicts on PR <number>`.

Tapping the same ticket again, or picking the same pull request prompt again,
reuses its window.

## Requirements

`tmux`, `treehouse`, `claude`, and `gh` must be on `PATH`. Each repo needs a
treehouse pool; run `treehouse init` in the repo first. `gh` must be
authenticated; run `gh auth login` once.

## Build and run

```bash
go build -o tmux-web .
./tmux-web
```

The server writes a default config on first start. Edit it, then restart.
Open `http://<host>:8080` on your phone.

Flags: `-config` (a config path) and `-listen` (a bind address).

## Configure

The config is TOML at `~/.config/tmux-web/config.toml`.

```toml
listen = ":8080"
prompt = ""
linear_api_key = ""
search_dirs = ["~/Code"]

[[group]]
name = "Work"

  [[group.repo]]
  name = "api"
  path = "/home/you/Code/work/api"
  linear_team = "API"
```

- `listen`: bind address. `:8080` binds all interfaces.
- `prompt`: default prompt in the new-window form.
- `linear_api_key`: a Linear personal API key. Blank hides the tickets list.
  The `LINEAR_API_KEY` environment variable overrides it.
- `search_dirs`: base directories to scan for more projects. The picker lists
  each immediate subdirectory. A picked project needs no treehouse pool.
- `[[group]]`: a heading in the repo picker.
- `[[group.repo]]`: a project. `linear_team` maps a Linear team key to the repo.

## Run as a systemd user service

The unit file is `systemd/tmux-web.service`. It runs as your user, so it shares
your tmux server.

```bash
go build -o ~/.local/bin/tmux-web .
ln -sf "$PWD/systemd/tmux-web.service" ~/.config/systemd/user/tmux-web.service
systemctl --user daemon-reload
systemctl --user enable --now tmux-web.service
```

To keep the service running after logout, enable lingering once:

```bash
loginctl enable-linger "$USER"
```

## Security

There is no authentication. Bind it to `localhost` or a trusted LAN only. Do
not expose it to the public internet. The server runs `claude`, `tmux`,
`treehouse`, and `gh` as your user.

## License

MIT. See [LICENSE](LICENSE).
