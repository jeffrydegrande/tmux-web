package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// PullRequest is one open GitHub pull request in a repo.
type PullRequest struct {
	Number int // e.g. 123
	Title  string
	Author string // GitHub login of the author
	Branch string // head branch name
}

// ListPullRequests returns the viewer's open pull requests for the repo at dir.
// It runs `gh pr list --author @me`, so gh must be on PATH and authenticated. A
// non-empty label limits the result to pull requests that carry that label. It
// returns an error when the repo has no GitHub remote or gh is not
// authenticated.
func ListPullRequests(dir, label string) ([]PullRequest, error) {
	args := []string{"pr", "list",
		"--state", "open", "--author", "@me", "--limit", "50",
		"--json", "number,title,author,headRefName"}
	if label != "" {
		args = append(args, "--label", label)
	}
	cmd := exec.Command("gh", args...)
	cmd.Dir = dir

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gh pr list: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return parsePulls([]byte(stdout.String()))
}

// parsePulls maps the JSON from `gh pr list --json ...` to pull requests.
func parsePulls(data []byte) ([]PullRequest, error) {
	var nodes []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
		HeadRefName string `json:"headRefName"`
	}
	if err := json.Unmarshal(data, &nodes); err != nil {
		return nil, fmt.Errorf("parse gh output: %w", err)
	}

	pulls := make([]PullRequest, 0, len(nodes))
	for _, n := range nodes {
		pulls = append(pulls, PullRequest{
			Number: n.Number,
			Title:  n.Title,
			Author: n.Author.Login,
			Branch: n.HeadRefName,
		})
	}
	return pulls, nil
}

// SignOff posts the signoff status check for a pull request in the repo at dir.
// It resolves the pull request head commit, then runs `gh signoff create` on
// that commit. The head commit is on the remote, so no checkout is needed. It
// returns an error when gh is not authenticated or the pull request is unknown.
func SignOff(dir string, prNumber int) error {
	if prNumber <= 0 {
		return fmt.Errorf("invalid pr number %d", prNumber)
	}
	sha, err := prHeadSHA(dir, prNumber)
	if err != nil {
		return err
	}

	cmd := exec.Command("gh", "signoff", "create", "--commit", sha)
	cmd.Dir = dir
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh signoff create: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// prHeadSHA returns the head commit sha of a pull request. It runs
// `gh pr view <n> --json headRefOid`.
func prHeadSHA(dir string, prNumber int) (string, error) {
	cmd := exec.Command("gh", "pr", "view", fmt.Sprint(prNumber), "--json", "headRefOid")
	cmd.Dir = dir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh pr view: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return parseHeadRefOid([]byte(stdout.String()))
}

// parseHeadRefOid reads the head commit sha from `gh pr view --json headRefOid`.
func parseHeadRefOid(data []byte) (string, error) {
	var v struct {
		HeadRefOid string `json:"headRefOid"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return "", fmt.Errorf("parse gh output: %w", err)
	}
	if v.HeadRefOid == "" {
		return "", fmt.Errorf("gh returned no head commit")
	}
	return v.HeadRefOid, nil
}

// repoPulls holds the open pull requests for one repo. Error is set when the gh
// call for the repo failed.
type repoPulls struct {
	RepoID   string
	RepoName string
	Pulls    []PullRequest
	Error    string
}

// PullRequestsByRepo lists open pull requests for every configured repo. It runs
// the gh calls in parallel. It returns one entry per repo that has open pull
// requests or that failed. It drops repos with no pull requests and no error.
func (c *Config) PullRequestsByRepo() []repoPulls {
	type job struct {
		id, name, path string
	}
	var jobs []job
	for gi := range c.Groups {
		for ri := range c.Groups[gi].Repos {
			repo := &c.Groups[gi].Repos[ri]
			jobs = append(jobs, job{id: repo.ID, name: repo.Name, path: repo.Path})
		}
	}

	results := make([]repoPulls, len(jobs))
	done := make(chan int, len(jobs))
	for i, j := range jobs {
		go func(i int, j job) {
			r := repoPulls{RepoID: j.id, RepoName: j.name}
			pulls, err := ListPullRequests(j.path, c.PRLabel)
			if err != nil {
				r.Error = err.Error()
			} else {
				r.Pulls = pulls
			}
			results[i] = r
			done <- i
		}(i, j)
	}
	for range jobs {
		<-done
	}

	out := make([]repoPulls, 0, len(results))
	for _, r := range results {
		if len(r.Pulls) == 0 && r.Error == "" {
			continue
		}
		out = append(out, r)
	}
	return out
}
