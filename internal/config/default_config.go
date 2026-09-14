package config

// defaultConfig is written to the XDG config path at first startup. Edit the
// file after that; the server never overwrites an existing config.
const defaultConfig = `# tmux-web configuration.

# Bind address. ":8080" binds all interfaces, so a phone on the same network
# can reach it. Use "127.0.0.1:8080" to keep it local.
listen = ":8080"

# Default prompt in the new-window form. Blank starts Claude Code with no
# prompt.
prompt = ""

# Linear personal API key. Blank hides the tickets list. The LINEAR_API_KEY
# environment variable overrides this field.
linear_api_key = ""

# Base directories to scan for more projects. The picker lists each immediate
# subdirectory. A leading "~" expands to your home directory. A picked project
# starts a session named after its basename. It needs no treehouse pool.
search_dirs = ["~/Code"]

# Filter the pull requests list to a label. Blank lists every open pull request
# you own. Set it to the label your CI gate adds when a deep review is required,
# e.g. "needs-team-review".
pr_label = ""

# Each [[group]] is a heading in the repo picker. Each [[group.repo]] is a
# project. "id" and "name" default from the path. Each repo needs a treehouse
# pool; run "treehouse init" in the repo first.
#
# "linear_team" is the Linear team key that maps to the repo. A ticket started
# from the tickets list opens in the repo whose "linear_team" matches the
# ticket team key. The team key is the prefix of the ticket id, e.g. "ENG" in
# "ENG-123". Two repos can not share a team.

[[group]]
name = "Work"

  [[group.repo]]
  name = "api"
  path = "/home/you/Code/work/api"
  linear_team = "API"

  [[group.repo]]
  name = "web"
  path = "/home/you/Code/work/web"
  linear_team = "WEB"

[[group]]
name = "Personal"

  [[group.repo]]
  name = "notes"
  path = "/home/you/Code/personal/notes"
`
