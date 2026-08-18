package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Repo is one project a new window can target. Each repo maps to its own tmux
// session, named after the directory. The repo must have a treehouse pool.
type Repo struct {
	// ID is the stable key used in URLs and forms. It defaults to the session
	// name when the config omits it.
	ID string `toml:"id"`
	// Name is the label shown in the UI. It defaults to the directory basename.
	Name string `toml:"name"`
	// Path is the absolute repo directory. treehouse runs here.
	Path string `toml:"path"`
	// Team is the Linear team key that maps to this repo. A ticket started from
	// the tickets list opens in the repo whose Team matches the ticket team
	// key. The team key is the prefix of a Linear identifier, e.g. "ENG" in
	// "ENG-123". The match ignores case. Two repos can not share a team.
	Team string `toml:"linear_team"`
}

// Group is a named set of repos. The UI shows repos split by group.
type Group struct {
	Name  string `toml:"name"`
	Repos []Repo `toml:"repo"`
}

// Config holds the server settings.
type Config struct {
	// Listen is the bind address. Default ":8080".
	Listen string `toml:"listen"`
	// Prompt is the default Claude Code prompt. The form can override it.
	Prompt string `toml:"prompt"`
	// LinearAPIKey is a Linear personal API key. It enables the tickets list.
	// The LINEAR_API_KEY environment variable overrides it.
	LinearAPIKey string `toml:"linear_api_key"`
	// Groups holds the projects a new window can target, split by group.
	Groups []Group `toml:"group"`
	// SearchDirs holds base directories to scan for more projects. The picker
	// lists each immediate subdirectory. A leading "~" expands to the home
	// directory. A picked directory starts a session named after its basename.
	// It needs no treehouse pool.
	SearchDirs []string `toml:"search_dirs"`
	// PRLabel filters the pull requests list to pull requests that carry this
	// label. Blank lists every open pull request. Set it to the label your CI
	// gate adds when a deep review is required, e.g. "needs-team-review".
	PRLabel string `toml:"pr_label"`
}

// Session returns the tmux session name for a repo. tmux treats "." and ":" as
// target separators, so they can not appear in a session name.
func (r Repo) Session() string {
	base := filepath.Base(r.Path)
	return strings.NewReplacer(".", "_", ":", "_").Replace(base)
}

// LoadConfig reads and validates the TOML config file.
func LoadConfig(path string) (*Config, error) {
	var c Config
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if c.Listen == "" {
		c.Listen = ":8080"
	}

	count := 0
	seen := map[string]bool{}
	teamOwner := map[string]string{}
	for gi := range c.Groups {
		group := &c.Groups[gi]
		if group.Name == "" {
			return nil, fmt.Errorf("group %d has no name", gi)
		}
		for ri := range group.Repos {
			repo := &group.Repos[ri]
			if repo.Path == "" {
				return nil, fmt.Errorf("group %q: repo %d has no path", group.Name, ri)
			}
			abs, err := filepath.Abs(repo.Path)
			if err != nil {
				return nil, fmt.Errorf("repo %q: %w", repo.Path, err)
			}
			repo.Path = abs
			if repo.Name == "" {
				repo.Name = filepath.Base(abs)
			}
			if repo.ID == "" {
				repo.ID = repo.Session()
			}
			if seen[repo.ID] {
				return nil, fmt.Errorf("duplicate repo id %q", repo.ID)
			}
			seen[repo.ID] = true

			if key := strings.ToUpper(strings.TrimSpace(repo.Team)); key != "" {
				if other, ok := teamOwner[key]; ok {
					return nil, fmt.Errorf("team %q maps to repos %q and %q", key, other, repo.ID)
				}
				teamOwner[key] = repo.ID
				repo.Team = key
			}
			count++
		}
	}
	if count == 0 {
		return nil, fmt.Errorf("config lists no repos")
	}

	for i, dir := range c.SearchDirs {
		abs, err := expandPath(dir)
		if err != nil {
			return nil, fmt.Errorf("search_dirs %q: %w", dir, err)
		}
		c.SearchDirs[i] = abs
	}
	return &c, nil
}

// expandPath expands a leading "~" to the home directory, then returns the
// absolute, cleaned path.
func expandPath(p string) (string, error) {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, strings.TrimPrefix(p, "~"))
	}
	return filepath.Abs(p)
}

// RepoByID finds a configured repo across all groups. It returns nil when no
// repo matches.
func (c *Config) RepoByID(id string) *Repo {
	for gi := range c.Groups {
		for ri := range c.Groups[gi].Repos {
			if c.Groups[gi].Repos[ri].ID == id {
				return &c.Groups[gi].Repos[ri]
			}
		}
	}
	return nil
}

// RepoByTeam finds the repo that owns a Linear team key. The match ignores
// case. It returns nil when no repo lists the team.
func (c *Config) RepoByTeam(key string) *Repo {
	key = strings.ToUpper(strings.TrimSpace(key))
	if key == "" {
		return nil
	}
	for gi := range c.Groups {
		for ri := range c.Groups[gi].Repos {
			repo := &c.Groups[gi].Repos[ri]
			if repo.Team == key {
				return repo
			}
		}
	}
	return nil
}

// ConfigPath resolves the config file path. A non-empty flag value wins.
// Otherwise it uses the XDG config directory: $XDG_CONFIG_HOME/tmux-web or
// ~/.config/tmux-web.
func ConfigPath(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate config dir: %w", err)
	}
	return filepath.Join(dir, "tmux-web", "config.toml"), nil
}

// EnsureConfig writes a default config when path does not exist. It returns
// true when it created the file.
func EnsureConfig(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(path, []byte(defaultConfig), 0o644); err != nil {
		return false, err
	}
	return true, nil
}
