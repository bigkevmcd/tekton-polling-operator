package git

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	pollingv1alpha1 "github.com/bigkevmcd/tekton-polling-operator/pkg/apis/polling/v1alpha1"
)

type GitHubPoller struct {
	client    *http.Client
	endpoint  string
	authToken string
}

const (
	chitauriPreview = "application/vnd.github.chitauri-preview+sha"
)

// NewGitHub creates a new GitHub poller.
func NewGitHub(c *http.Client, authToken string) *GitHubPoller {
	return &GitHubPoller{client: c, endpoint: "https://api.github.com", authToken: authToken}
}

func (g GitHubPoller) Poll(repo string, pr *pollingv1alpha1.RepositoryStatus) (*pollingv1alpha1.RepositoryStatus, error) {
	requestURL, err := makeURL(g.endpoint, repo, pr.Ref)
	if err != nil {
		return nil, fmt.Errorf("failed to make the request URL: %w", err)
	}
	req, err := http.NewRequest("GET", requestURL, nil)
	if pr.ETag != "" {
		req.Header.Add("If-None-Match", pr.ETag)
	}
	req.Header.Add("Accept", chitauriPreview)
	if g.authToken != "" {
		req.Header.Add("Authorization", fmt.Sprintf("token %s", g.authToken))
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get current commit: %v", err)
	}
	// TODO: Return an error type that we can identify as a NotFound, likely
	// this is either a security token issue, or an unknown repo.
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("server error: %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotModified {
		return pr, nil
	}
	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	var gc githubCommit
	err = dec.Decode(&gc)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}
	return &pollingv1alpha1.RepositoryStatus{Ref: pr.Ref, SHA: gc.SHA, ETag: resp.Header.Get("ETag")}, nil
}

func makeURL(endpoint, repo, ref string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	parsed.Path = path.Join("repos", repo, "commits", ref)
	return parsed.String(), nil
}

type githubCommit struct {
	SHA string `json:"sha"`
}
