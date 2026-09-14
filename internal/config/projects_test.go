package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchProjects(t *testing.T) {
	base := t.TempDir()
	for _, name := range []string{"api", "apiary", "myapi", "web", ".hidden"} {
		if err := os.Mkdir(filepath.Join(base, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// A file must not appear as a project.
	if err := os.WriteFile(filepath.Join(base, "readme"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{SearchDirs: []string{base, "/does/not/exist"}}

	// A blank query returns nothing.
	if hits := cfg.SearchProjects("", 25); hits != nil {
		t.Errorf("blank query returned %d hits, want 0", len(hits))
	}

	// Prefix matches rank before mid-string matches, then sort by name.
	hits := cfg.SearchProjects("api", 25)
	got := []string{}
	for _, h := range hits {
		got = append(got, h.Name)
	}
	want := []string{"api", "apiary", "myapi"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("hits = %v, want %v", got, want)
	}
	if hits[0].Value != ProjectPrefix+filepath.Join(base, "api") {
		t.Errorf("value = %q", hits[0].Value)
	}

	// A miss returns an empty list. The limit caps the result.
	if hits := cfg.SearchProjects("zzz", 25); len(hits) != 0 {
		t.Errorf("miss returned %d hits, want 0", len(hits))
	}
	if hits := cfg.SearchProjects("api", 1); len(hits) != 1 {
		t.Errorf("limit 1 returned %d hits, want 1", len(hits))
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
