package git

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	pollingv1 "github.com/bigkevmcd/tekton-polling-operator/pkg/apis/polling/v1alpha1"
)

// TODO: add logging - especially of the response body.

type GitLabPoller struct {
	client    *http.Client
	endpoint  string
	authToken string
}

// NewGitLabPoller creates a new GitLab poller.
func NewGitLabPoller(c *http.Client, authToken string) *GitLabPoller {
	return &GitLabPoller{client: c, endpoint: "https://gitlab.com", authToken: authToken}
}

func (g GitLabPoller) Poll(repo string, pr pollingv1.PollStatus) (pollingv1.PollStatus, error) {
	requestURL := makeGitLabURL(g.endpoint, repo, pr.Ref)
	req, err := http.NewRequest("GET", requestURL, nil)
	if pr.ETag != "" {
		req.Header.Add("If-None-Match", pr.ETag)
	}
	req.Header.Add("Accept", chitauriPreview)
	if g.authToken != "" {
		req.Header.Add("Private-Token", g.authToken)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return pollingv1.PollStatus{}, fmt.Errorf("failed to get current commit: %v", err)
	}
	// TODO: Return an error type that we can identify as a NotFound, likely
	// this is either a security token issue, or an unknown repo.
	if resp.StatusCode >= http.StatusBadRequest {
		return pollingv1.PollStatus{}, fmt.Errorf("server error: %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotModified {
		return pr, nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	var gc []gitlabCommit
	err = json.Unmarshal(body, &gc)
	if err != nil {
		return pollingv1.PollStatus{}, fmt.Errorf("failed to decode response body: %w", err)
	}
	return pollingv1.PollStatus{Ref: pr.Ref, SHA: gc[0].ID, ETag: resp.Header.Get("ETag")}, nil
}

func makeGitLabURL(endpoint, repo, ref string) string {
	values := url.Values{
		"ref": []string{ref},
	}
	return fmt.Sprintf("%s/api/v4/projects/%s/repository/commits?%s",
		endpoint, strings.Replace(repo, "/", "%2F", -1),
		values.Encode())
}

type gitlabCommit struct {
	ID string `json:"id"`
}
