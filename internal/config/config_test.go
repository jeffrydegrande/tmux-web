package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfigGroups(t *testing.T) {
	path := writeTemp(t, `
listen = ":9000"
prompt = "go"

[[group]]
name = "Acme"
  [[group.repo]]
  path = "/home/x/Code/acme/api"
  [[group.repo]]
  name = "web"
  path = "/home/x/Code/acme/web-server"

[[group]]
name = "Personal"
  [[group.repo]]
  path = "/home/x/Code/personal/notes"
`)

	c, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Listen != ":9000" {
		t.Errorf("listen = %q", c.Listen)
	}
	if len(c.Groups) != 2 {
		t.Fatalf("groups = %d, want 2", len(c.Groups))
	}

	// id and name default from the path.
	api := c.RepoByID("api")
	if api == nil || api.Name != "api" {
		t.Fatalf("api lookup failed: %+v", api)
	}
	// explicit name is kept; id still defaults from the session.
	if r := c.RepoByID("web-server"); r == nil || r.Name != "web" {
		t.Fatalf("web lookup failed: %+v", r)
	}
	if c.RepoByID("missing") != nil {
		t.Error("missing id should return nil")
	}
}

func TestRepoByTeam(t *testing.T) {
	path := writeTemp(t, `
[[group]]
name = "Acme"
  [[group.repo]]
  path = "/home/x/Code/acme/api"
  linear_team = "API"
  [[group.repo]]
  path = "/home/x/Code/acme/web-server"
  linear_team = "web"
`)
	c, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	// The team key maps to its repo.
	if r := c.RepoByTeam("API"); r == nil || r.ID != "api" {
		t.Fatalf("API lookup failed: %+v", r)
	}
	// The match ignores case, in the config and in the query.
	if r := c.RepoByTeam("web"); r == nil || r.ID != "web-server" {
		t.Fatalf("web lookup failed: %+v", r)
	}
	if r := c.RepoByTeam("WEB"); r == nil || r.ID != "web-server" {
		t.Fatalf("WEB lookup failed: %+v", r)
	}
	// An unmapped team returns nil.
	if c.RepoByTeam("NONE") != nil {
		t.Error("unmapped team should return nil")
	}
	if c.RepoByTeam("") != nil {
		t.Error("empty team should return nil")
	}
}

func TestLoadConfigDuplicateTeam(t *testing.T) {
	path := writeTemp(t, `
[[group]]
name = "A"
  [[group.repo]]
  path = "/tmp/one"
  linear_team = "API"
  [[group.repo]]
  path = "/tmp/two"
  linear_team = "api"
`)
	if _, err := LoadConfig(path); err == nil {
		t.Error("want duplicate team error, got nil")
	}
}

func TestLoadConfigDuplicateID(t *testing.T) {
	path := writeTemp(t, `
[[group]]
name = "A"
  [[group.repo]]
  path = "/tmp/dup"
[[group]]
name = "B"
  [[group.repo]]
  path = "/other/dup"
`)
	if _, err := LoadConfig(path); err == nil {
		t.Error("want duplicate id error, got nil")
	}
}

func TestLoadConfigNoRepos(t *testing.T) {
	path := writeTemp(t, `listen = ":8080"`)
	if _, err := LoadConfig(path); err == nil {
		t.Error("want no-repos error, got nil")
	}
}

func TestEnsureConfigWritesDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.toml")
	created, err := EnsureConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("want created = true")
	}
	// The default must load and parse.
	if _, err := LoadConfig(path); err != nil {
		t.Fatalf("default config does not load: %v", err)
	}
	// A second call must not report a new write.
	if created, _ := EnsureConfig(path); created {
		t.Error("second EnsureConfig reported a write")
	}
}
