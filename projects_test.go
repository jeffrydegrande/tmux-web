package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverProjects(t *testing.T) {
	base := t.TempDir()
	for _, name := range []string{"api", "web", ".hidden"} {
		if err := os.Mkdir(filepath.Join(base, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// A file must not appear as a project.
	if err := os.WriteFile(filepath.Join(base, "readme"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{SearchDirs: []string{base, "/does/not/exist"}}
	groups := cfg.DiscoverProjects()
	if len(groups) != 1 {
		t.Fatalf("groups = %d, want 1 (missing dir skipped)", len(groups))
	}
	got := []string{}
	for _, p := range groups[0].Projects {
		got = append(got, p.Name)
	}
	// Sorted, no hidden dir, no file.
	want := []string{"api", "web"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("projects = %v, want %v", got, want)
	}
	if p := groups[0].Projects[0]; p.Value != projectPrefix+filepath.Join(base, "api") {
		t.Errorf("value = %q", p.Value)
	}
}

func TestProjectRepo(t *testing.T) {
	base := t.TempDir()
	proj := filepath.Join(base, "api")
	if err := os.Mkdir(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{SearchDirs: []string{base}}

	// A valid immediate subdirectory maps to a repo.
	if r := cfg.ProjectRepo(proj); r == nil || r.Name != "api" || r.Path != proj {
		t.Fatalf("valid project failed: %+v", r)
	}
	// A path outside the search dirs is rejected.
	if r := cfg.ProjectRepo("/etc"); r != nil {
		t.Error("path outside search dirs must be rejected")
	}
	// A traversal that escapes the base is rejected.
	if r := cfg.ProjectRepo(filepath.Join(base, "..", "elsewhere")); r != nil {
		t.Error("traversal must be rejected")
	}
	// A missing subdirectory is rejected.
	if r := cfg.ProjectRepo(filepath.Join(base, "ghost")); r != nil {
		t.Error("missing dir must be rejected")
	}
}
