package main

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
)

// Project is one directory found under a search dir. A picked project starts a
// tmux session named after its basename.
type Project struct {
	// Name is the directory basename, shown in the picker.
	Name string
	// Value is the option value in the form. It carries the path with a "dir:"
	// prefix, so the window handler can tell it apart from a configured repo id.
	Value string
}

// ProjectGroup holds the projects found under one search dir. Base is the label
// shown in the picker, e.g. "~/Code".
type ProjectGroup struct {
	Base     string
	Projects []Project
}

// projectPrefix marks a form value as a search-dir path, not a repo id.
const projectPrefix = "dir:"

// DiscoverProjects scans each search dir for immediate subdirectories. It
// returns one group per search dir that holds at least one subdirectory. It
// skips a search dir that does not exist or can not be read. The lists are
// sorted by name.
func (c *Config) DiscoverProjects() []ProjectGroup {
	home, _ := os.UserHomeDir()

	var groups []ProjectGroup
	for _, base := range c.SearchDirs {
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		var projects []Project
		for _, e := range entries {
			if !e.IsDir() || e.Name() == "" || e.Name()[0] == '.' {
				continue
			}
			path := filepath.Join(base, e.Name())
			projects = append(projects, Project{
				Name:  e.Name(),
				Value: projectPrefix + path,
			})
		}
		if len(projects) == 0 {
			continue
		}
		sort.Slice(projects, func(i, j int) bool { return projects[i].Name < projects[j].Name })
		groups = append(groups, ProjectGroup{Base: shortenHome(base, home), Projects: projects})
	}
	return groups
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
