package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// linearEndpoint is the Linear GraphQL API.
const linearEndpoint = "https://api.linear.app/graphql"

// Issue is one Linear ticket assigned to the viewer.
type Issue struct {
	Identifier string // e.g. "ENG-123"
	Title      string
	State      string // workflow state name, e.g. "In Progress"
	TeamKey    string // Linear team key, e.g. "ENG"
}

// LinearClient calls the Linear API with a personal API key.
type LinearClient struct {
	apiKey string
	http   *http.Client
}

// NewLinearClient returns a client, or nil when no API key is set.
func NewLinearClient(apiKey string) *LinearClient {
	if apiKey == "" {
		return nil
	}
	return &LinearClient{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 12 * time.Second},
	}
}

// assignedIssuesQuery fetches the viewer's assigned tickets, newest first.
const assignedIssuesQuery = `query {
  viewer {
    assignedIssues(first: 50, orderBy: updatedAt) {
      nodes { identifier title state { name type } team { key } }
    }
  }
}`

// Issues returns the viewer's open assigned tickets. It drops completed and
// canceled tickets. It also drops tickets that are in review; the GitHub pull
// requests list covers those instead.
func (c *LinearClient) Issues() ([]Issue, error) {
	body, _ := json.Marshal(map[string]string{"query": assignedIssuesQuery})
	req, err := http.NewRequest(http.MethodPost, linearEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("linear api: %s: %s", resp.Status, truncate(string(data), 200))
	}

	var out struct {
		Data struct {
			Viewer struct {
				AssignedIssues struct {
					Nodes []struct {
						Identifier string `json:"identifier"`
						Title      string `json:"title"`
						State      struct {
							Name string `json:"name"`
							Type string `json:"type"`
						} `json:"state"`
						Team struct {
							Key string `json:"key"`
						} `json:"team"`
					} `json:"nodes"`
				} `json:"assignedIssues"`
			} `json:"viewer"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse linear response: %w", err)
	}
	if len(out.Errors) > 0 {
		return nil, fmt.Errorf("linear api: %s", out.Errors[0].Message)
	}

	var issues []Issue
	for _, n := range out.Data.Viewer.AssignedIssues.Nodes {
		if !keepIssue(n.State.Type, n.State.Name) {
			continue
		}
		issues = append(issues, Issue{
			Identifier: n.Identifier,
			Title:      n.Title,
			State:      n.State.Name,
			TeamKey:    n.Team.Key,
		})
	}
	return issues, nil
}

// keepIssue reports whether a ticket belongs in the list. It drops completed
// and canceled tickets. It also drops tickets in review, because the GitHub
// pull requests list covers those. Workspaces name the review state
// differently, so the match is by substring.
func keepIssue(stateType, stateName string) bool {
	if stateType == "completed" || stateType == "canceled" {
		return false
	}
	if strings.Contains(strings.ToLower(stateName), "review") {
		return false
	}
	return true
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
