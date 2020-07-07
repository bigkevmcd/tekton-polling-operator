package git

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	pollingv1alpha1 "github.com/bigkevmcd/tekton-polling-operator/pkg/apis/polling/v1alpha1"
)

const testToken = "test12345"

var _ CommitPoller = (*GitHubPoller)(nil)

func TestWithUnknownETag(t *testing.T) {
	etag := `W/"878f43039ad0553d0d3122d8bc171b01"`
	as := makeGitHubAPIServer(t, testToken, "/repos/testing/repo/commits/master", etag, mustReadFile(t, "testdata/github_commit.json"))
	t.Cleanup(as.Close)
	g := NewGitHubPoller(as.Client(), testToken)
	g.endpoint = as.URL

	polled, err := g.Poll("testing/repo", pollingv1alpha1.PollStatus{Ref: "master"})
	if err != nil {
		t.Fatal(err)
	}

	if polled.ETag != etag {
		t.Errorf("Poll() ETag got %s, want %s", polled.ETag, etag)
	}
	if polled.SHA != "7638417db6d59f3c431d3e1f261cc637155684cd" {
		t.Errorf("Poll() SHA got %s, want %s", polled.SHA, "7638417db6d59f3c431d3e1f261cc637155684cd")
	}
}

func TestWithKnownTag(t *testing.T) {
	etag := `W/"878f43039ad0553d0d3122d8bc171b01"`
	as := makeGitHubAPIServer(t, testToken, "/repos/testing/repo/commits/master", etag, nil)
	t.Cleanup(as.Close)
	g := NewGitHubPoller(as.Client(), testToken)
	g.endpoint = as.URL

	polled, err := g.Poll("testing/repo", pollingv1alpha1.PollStatus{Ref: "master", ETag: etag})
	if err != nil {
		t.Fatal(err)
	}

	if polled.ETag != etag {
		t.Fatalf("Poll() got %s, want %s", polled.ETag, etag)
	}
}

func TestWithNotFoundResponse(t *testing.T) {
	etag := `W/"878f43039ad0553d0d3122d8bc171b01"`
	as := makeGitHubAPIServer(t, testToken, "/repos/testing/repo/commits/master", etag, nil)
	t.Cleanup(as.Close)
	g := NewGitHubPoller(as.Client(), testToken)
	g.endpoint = as.URL

	_, err := g.Poll("testing/testing", pollingv1alpha1.PollStatus{Ref: "master", ETag: etag})
	if err.Error() != "server error: 404" {
		t.Fatal(err)
	}
}

// It's impossible to distinguish between unknown repo, and bad auth token, both
// respond with a 404.
func TestWithBadAuthentication(t *testing.T) {
	etag := `W/"878f43039ad0553d0d3122d8bc171b01"`
	as := makeGitHubAPIServer(t, testToken, "/repos/testing/repo/commits/master", etag, nil)
	t.Cleanup(as.Close)
	g := NewGitHubPoller(as.Client(), "anotherToken")
	g.endpoint = as.URL

	_, err := g.Poll("testing/repo", pollingv1alpha1.PollStatus{Ref: "master", ETag: etag})
	if err.Error() != "server error: 404" {
		t.Fatal(err)
	}
}

// With no auth-token, no auth header should be sent.
func TestWithNoAuthentication(t *testing.T) {
	etag := `W/"878f43039ad0553d0d3122d8bc171b01"`
	as := makeGitHubAPIServer(t, "", "/repos/testing/repo/commits/master", etag, nil)
	t.Cleanup(as.Close)
	g := NewGitHubPoller(as.Client(), "")
	g.endpoint = as.URL

	_, err := g.Poll("testing/repo", pollingv1alpha1.PollStatus{Ref: "master", ETag: etag})
	if err != nil {
		t.Fatal(err)
	}
}

// makeAPIServer is used during testing to create an HTTP server to return
// fixtures if the request matches.
func makeGitHubAPIServer(t *testing.T, authToken, wantPath, etag string, response []byte) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if authToken != "" {
			if auth := r.Header.Get("Authorization"); auth != fmt.Sprintf("token %s", authToken) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}
		if auth := r.Header.Get("Authorization"); auth != "" && authToken == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if etag == r.Header.Get("If-None-Match") {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if r.Header.Get("Accept") != chitauriPreview {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
		w.Header().Set("ETag", etag)
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	}))
}

func mustReadFile(t *testing.T, filename string) []byte {
	t.Helper()
	d, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
