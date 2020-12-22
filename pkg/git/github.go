package git

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	pollingv1 "github.com/bigkevmcd/tekton-polling-operator/pkg/apis/polling/v1alpha1"
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

// NewGitHubPoller creates and returns a new GitHub poller.
func NewGitHubPoller(c *http.Client, endpoint, authToken string) *GitHubPoller {
	return &GitHubPoller{client: c, endpoint: endpoint, authToken: authToken}
}

func (g GitHubPoller) Poll(repo string, pr pollingv1.PollStatus) (pollingv1.PollStatus, Commit, error) {
	requestURL, err := makeGitHubURL(g.endpoint, repo, pr.Ref)
	if err != nil {
		return pollingv1.PollStatus{}, nil, fmt.Errorf("failed to make the request URL: %w", err)
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
		return pollingv1.PollStatus{}, nil, fmt.Errorf("failed to get current commit: %v", err)
	}
	// TODO: Return an error type that we can identify as a NotFound, likely
	// this is either a security token issue, or an unknown repo.
	if resp.StatusCode >= http.StatusBadRequest {
		return pollingv1.PollStatus{}, nil, fmt.Errorf("server error: %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotModified {
		return pr, nil, nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	var gc map[string]interface{}
	err = json.Unmarshal(body, &gc)
	if err != nil {
		return pollingv1.PollStatus{}, nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	sha := gc["sha"].(string)
	tag, err := g.findTags(repo, sha)
	if err != nil {
		return pollingv1.PollStatus{}, nil, fmt.Errorf("failed to get current commit: %v", err)
	}
	gc["tag"] = tag

	return pollingv1.PollStatus{Ref: pr.Ref, SHA: sha, Tag: tag, ETag: resp.Header.Get("ETag")}, gc, nil
}

type TagEntry struct {
	Name       string       `json:"name,omitempty"`
	ZipballURL string       `json:"zipball_url,omitempty"`
	TarballURL string       `json:"tarball_url,omitempty"`
	NodeID     string       `json:"node_id,omitempty"`
	Commit     CommitSHAUrl `json:"commit,omitempty"`
}

type CommitSHAUrl struct {
	SHA string `json:"sha,omitempty"`
	URL string `json:"url,omitempty"`
}

func (g GitHubPoller) findTags(repo string, sha string) (string, error) {
	requestURL, err := makeGitHubTagURL(g.endpoint, repo)
	if err != nil {
		return sha, err
	}

	req, err := http.NewRequest("GET", requestURL, nil)

	req.Header.Add("Accept", chitauriPreview)
	if g.authToken != "" {
		req.Header.Add("Authorization", fmt.Sprintf("token %s", g.authToken))
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return sha, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return sha, fmt.Errorf("server error: %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return sha, err
	}

	var tags []TagEntry
	err = json.Unmarshal(body, &tags)
	if err != nil {
		return sha, fmt.Errorf("failed to decode response body: %w", err)
	}

	for _, tag := range tags {
		if sha == tag.Commit.SHA {
			return tag.Name, nil
		}
	}

	return sha, nil
}

func makeGitHubTagURL(endpoint, repo string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	parsed.Path = path.Join("repos", repo, "tags")
	return parsed.String(), nil
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
