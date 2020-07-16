package git

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	pollingv1alpha1 "github.com/bigkevmcd/tekton-polling-operator/api/v1alpha1"
)

// TODO: add logging - especially of the response body.

type GitHubPoller struct {
	client    *http.Client
	endpoint  string
	authToken string
}

const (
	chitauriPreview = "application/vnd.github.chitauri-preview+sha"
)

// NewGitHubPoller creates a new GitHub poller.
func NewGitHubPoller(c *http.Client, endpoint, authToken string) *GitHubPoller {
	if endpoint == "" {
		endpoint = "https://api.github.com"
	}
	return &GitHubPoller{client: c, endpoint: endpoint, authToken: authToken}
}

func (g GitHubPoller) Poll(repo string, pr pollingv1alpha1.PollStatus) (pollingv1alpha1.PollStatus, error) {
	requestURL, err := makeGitHubURL(g.endpoint, repo, pr.Ref)
	if err != nil {
		return pollingv1alpha1.PollStatus{}, fmt.Errorf("failed to make the request URL: %w", err)
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
		return pollingv1alpha1.PollStatus{}, fmt.Errorf("failed to get current commit: %v", err)
	}
	// TODO: Return an error type that we can identify as a NotFound, likely
	// this is either a security token issue, or an unknown repo.
	if resp.StatusCode >= http.StatusBadRequest {
		return pollingv1alpha1.PollStatus{}, fmt.Errorf("server error: %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotModified {
		return pr, nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	var gc githubCommit
	err = json.Unmarshal(body, &gc)
	if err != nil {
		return pollingv1alpha1.PollStatus{}, fmt.Errorf("failed to decode response body: %w", err)
	}
	return pollingv1alpha1.PollStatus{Ref: pr.Ref, SHA: gc.SHA, ETag: resp.Header.Get("ETag")}, nil
}

func makeGitHubURL(endpoint, repo, ref string) (string, error) {
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
