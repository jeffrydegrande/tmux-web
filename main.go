package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Server holds the shared state for the HTTP handlers.
type Server struct {
	cfg    *Config
	tmpl   *template.Template
	linear *LinearClient
}

func main() {
	configFlag := flag.String("config", "", "path to the config file (default: XDG config dir)")
	listen := flag.String("listen", "", "override the listen address")
	flag.Parse()

	configPath, err := ConfigPath(*configFlag)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if created, err := EnsureConfig(configPath); err != nil {
		log.Fatalf("config: %v", err)
	} else if created {
		log.Printf("wrote a default config to %s; edit it to match your repos", configPath)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if *listen != "" {
		cfg.Listen = *listen
	}

	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatalf("templates: %v", err)
	}

	apiKey := cfg.LinearAPIKey
	if env := os.Getenv("LINEAR_API_KEY"); env != "" {
		apiKey = env
	}

	srv := &Server{cfg: cfg, tmpl: tmpl, linear: NewLinearClient(apiKey)}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", srv.handleIndex)
	mux.HandleFunc("GET /sessions", srv.handleSessions)
	mux.HandleFunc("GET /tickets", srv.handleTickets)
	mux.HandleFunc("GET /pulls", srv.handlePulls)
	mux.HandleFunc("GET /projects", srv.handleProjects)
	mux.HandleFunc("GET /snapshot", srv.handleSnapshot)
	mux.HandleFunc("POST /windows", srv.handleNewWindow)
	mux.Handle("GET /static/", http.FileServer(http.FS(staticFS)))

	log.Printf("tmux-web listening on %s", cfg.Listen)
	if err := http.ListenAndServe(cfg.Listen, mux); err != nil {
		log.Fatal(err)
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	sessions, err := ListSessions()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, "index.html", map[string]any{
		"Groups":   s.cfg.Groups,
		"Sessions": sessions,
		"Prompt":   s.cfg.Prompt,
		"Linear":   s.linear != nil,
	})
}

func (s *Server) handleTickets(w http.ResponseWriter, r *http.Request) {
	if s.linear == nil {
		s.render(w, "tickets.html", map[string]any{
			"Error": "Set LINEAR_API_KEY to list tickets.",
		})
		return
	}
	issues, err := s.linear.Issues()
	if err != nil {
		s.render(w, "tickets.html", map[string]any{"Error": err.Error()})
		return
	}
	s.render(w, "tickets.html", map[string]any{"Groups": groupTickets(s.cfg, issues)})
}

// prActions are the prompts a pull request can start Claude Code with. The user
// picks the pull request first, then picks one of these.
var prActions = []struct {
	Label     string
	Suffix    string
	PromptFmt string
}{
	{"Deep review", "review", "deep review on PR %d"},
	{"Fix merge conflicts", "fix", "fix merge conflicts on PR %d"},
}

// pullActionView is one prompt choice for a pull request. Name is the window
// name. Prompt is the Claude Code prompt.
type pullActionView struct {
	Label  string
	Name   string
	Prompt string
}

// pullView is one pull request with its prompt choices. The list shows only the
// viewer's own pull requests, so it shows the branch, not the author.
type pullView struct {
	Number  int
	Title   string
	Branch  string
	Actions []pullActionView
}

// repoPullsView groups pull requests by repo for the template.
type repoPullsView struct {
	RepoID   string
	RepoName string
	Error    string
	Pulls    []pullView
}

// handlePulls lists open GitHub pull requests, grouped by repo. Each pull
// request offers a choice of prompts. The choice opens a window named after the
// pull request and the action.
func (s *Server) handlePulls(w http.ResponseWriter, r *http.Request) {
	repos := s.cfg.PullRequestsByRepo()
	views := make([]repoPullsView, 0, len(repos))
	for _, rp := range repos {
		v := repoPullsView{RepoID: rp.RepoID, RepoName: rp.RepoName, Error: rp.Error}
		for _, p := range rp.Pulls {
			pv := pullView{Number: p.Number, Title: p.Title, Branch: p.Branch}
			for _, a := range prActions {
				pv.Actions = append(pv.Actions, pullActionView{
					Label:  a.Label,
					Name:   fmt.Sprintf("pr-%d-%s", p.Number, a.Suffix),
					Prompt: fmt.Sprintf(a.PromptFmt, p.Number),
				})
			}
			v.Pulls = append(v.Pulls, pv)
		}
		views = append(views, v)
	}
	s.render(w, "pulls.html", map[string]any{"Repos": views, "Label": s.cfg.PRLabel})
}

// ticketView pairs a ticket with the repo its Linear team maps to. RepoID is
// empty when no repo lists the team. Then the ticket falls back to the repo
// selected in the form.
type ticketView struct {
	Issue
	RepoID   string
	RepoName string
}

// ticketGroup holds the tickets that map to one project. Tickets with no mapped
// team fall in a group named "Other".
type ticketGroup struct {
	RepoName string
	Issues   []ticketView
}

// groupTickets groups tickets by the repo their Linear team maps to. It keeps
// the order in which each group first appears.
func groupTickets(cfg *Config, issues []Issue) []*ticketGroup {
	var order []string
	byKey := map[string]*ticketGroup{}
	for _, issue := range issues {
		repo := cfg.RepoByTeam(issue.TeamKey)
		key, name := "", "Other"
		v := ticketView{Issue: issue}
		if repo != nil {
			key, name = repo.ID, repo.Name
			v.RepoID, v.RepoName = repo.ID, repo.Name
		}
		g, ok := byKey[key]
		if !ok {
			g = &ticketGroup{RepoName: name}
			byKey[key] = g
			order = append(order, key)
		}
		g.Issues = append(g.Issues, v)
	}
	groups := make([]*ticketGroup, 0, len(order))
	for _, k := range order {
		groups = append(groups, byKey[k])
	}
	return groups
}

// handleProjects returns projects that match the search query. It caps the
// result so a search dir with hundreds of projects stays usable.
func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	const limit = 25
	query := r.URL.Query().Get("q")
	hits := s.cfg.SearchProjects(query, limit)
	s.render(w, "project_results.html", map[string]any{
		"Query":  strings.TrimSpace(query),
		"Hits":   hits,
		"Capped": len(hits) == limit,
	})
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := ListSessions()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, "sessions.html", map[string]any{"Sessions": sessions})
}

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	text, err := CapturePane(target)
	if err != nil {
		s.render(w, "snapshot.html", map[string]any{
			"Target": target,
			"Error":  err.Error(),
		})
		return
	}
	s.render(w, "snapshot.html", map[string]any{
		"Target": target,
		"Text":   text,
	})
}

func (s *Server) handleNewWindow(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.fail(w, err)
		return
	}
	// The value is a configured repo id, or a "dir:" path from the project
	// picker. A project starts a session in its own directory with no worktree.
	value := r.FormValue("repo")
	worktree := true
	var repo *Repo
	if path, ok := strings.CutPrefix(value, projectPrefix); ok {
		repo = s.cfg.ProjectRepo(path)
		worktree = false
	} else {
		repo = s.cfg.RepoByID(value)
	}
	if repo == nil {
		http.Error(w, "unknown repo", http.StatusBadRequest)
		return
	}
	prompt := r.FormValue("prompt")

	// A ticket passes its identifier as the name so the window is idempotent.
	// A blank name means a plain new window with a generated name.
	name := sanitizeName(r.FormValue("name"))
	if name == "" {
		name = generateName()
	}

	reused, err := startWindow(repo, name, prompt, worktree)
	if err != nil {
		s.render(w, "new_result.html", map[string]any{"Error": err.Error()})
		return
	}

	sessions, err := ListSessions()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.render(w, "new_result.html", map[string]any{
		"Window":   name,
		"Session":  repo.Session(),
		"Reused":   reused,
		"Sessions": sessions,
	})
}

// startWindow runs the full flow: ensure the session, lease a worktree when the
// target uses one, then open a window that starts Claude Code. It returns true
// when a window with the same name already existed, so the worktree lease is
// left untouched. A project target (worktree false) runs Claude Code in the
// project directory itself.
func startWindow(repo *Repo, name, prompt string, worktree bool) (reused bool, err error) {
	if err := EnsureSession(repo.Session(), repo.Path); err != nil {
		return false, err
	}
	exists, err := HasWindow(repo.Session(), name)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}
	dir := repo.Path
	if worktree {
		dir, err = LeaseWorktree(repo.Path, name)
		if err != nil {
			return false, err
		}
	}
	return false, NewWindow(repo.Session(), name, dir, prompt)
}

// sanitizeName maps a name to tmux-safe characters. tmux uses "." and ":" as
// target separators, so they can not appear in a window name.
func sanitizeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

// generateName builds a unique, readable window name. A short random suffix
// avoids a collision when two windows start in the same second.
func generateName() string {
	var b [2]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("w-%s-%s", time.Now().Format("0102-1504"), hex.EncodeToString(b[:]))
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
	}
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
