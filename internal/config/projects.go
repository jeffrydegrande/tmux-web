package config

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// Project is one directory found under a search dir. A picked project starts a
// tmux session named after its basename.
type Project struct {
	// Name is the directory basename, shown in the result.
	Name string
	// Base is the search dir the project sits in, shortened with "~". It tells
	// apart two projects with the same name.
	Base string
	// Value is the form value. It carries the path with a "dir:" prefix, so the
	// window handler can tell it apart from a configured repo id.
	Value string
}

// ProjectPrefix marks a form value as a search-dir path, not a repo id.
const ProjectPrefix = "dir:"

// SearchProjects returns projects whose name matches the query. The match is a
// case-insensitive substring. A blank query returns nothing, because the search
// dirs can hold hundreds of projects. Names that start with the query rank
// first. The result is capped at limit.
func (c *Config) SearchProjects(query string, limit int) []Project {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}
	home, _ := os.UserHomeDir()

	var hits []Project
	for _, base := range c.SearchDirs {
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		short := shortenHome(base, home)
		for _, e := range entries {
			if !e.IsDir() || e.Name() == "" || e.Name()[0] == '.' {
				continue
			}
			if !strings.Contains(strings.ToLower(e.Name()), query) {
				continue
			}
			hits = append(hits, Project{
				Name:  e.Name(),
				Base:  short,
				Value: ProjectPrefix + filepath.Join(base, e.Name()),
			})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		pi := strings.HasPrefix(strings.ToLower(hits[i].Name), query)
		pj := strings.HasPrefix(strings.ToLower(hits[j].Name), query)
		if pi != pj {
			return pi // a prefix match ranks before a mid-string match
		}
		if hits[i].Name != hits[j].Name {
			return hits[i].Name < hits[j].Name
		}
		return hits[i].Base < hits[j].Base
	})
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

// ProjectRepo turns a validated search-dir path into a Repo. The session name
// is the basename, the same as a configured repo. It returns nil when path is
// not an immediate subdirectory of a configured search dir. This check stops a
// form from starting a session in any directory on disk.
func (c *Config) ProjectRepo(path string) *Repo {
	path = filepath.Clean(path)
	if !slices.Contains(c.SearchDirs, filepath.Dir(path)) {
		return nil
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		return nil
	}
	return &Repo{Path: path, Name: filepath.Base(path)}
}

// shortenHome replaces a home-directory prefix with "~" for display.
func shortenHome(path, home string) string {
	if home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if rel, err := filepath.Rel(home, path); err == nil && rel != ".." && !filepath.IsAbs(rel) &&
		(len(rel) < 2 || rel[:2] != "..") {
		return "~/" + rel
	}
	return path
}
