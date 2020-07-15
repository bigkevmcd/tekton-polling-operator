package git

import (
	"net/http"
	"net/http/httptest"
	"testing"

	pollingv1alpha1 "github.com/bigkevmcd/tekton-polling-operator/pkg/apis/polling/v1alpha1"
)

var _ CommitPoller = (*GitLabPoller)(nil)

func TestNewGitLabPoller(t *testing.T) {
	newTests := []struct {
		endpoint     string
		wantEndpoint string
	}{
		{"", "https://gitlab.com"},
		{"https://gl.example.com", "https://gl.example.com"},
	}

	for _, tt := range newTests {
		c := NewGitLabPoller(http.DefaultClient, tt.endpoint, "testToken")

		if c.endpoint != tt.wantEndpoint {
			t.Errorf("%#v got %#v, want %#v", tt.endpoint, c.endpoint, tt.wantEndpoint)
		}
	}
}

func TestGitLabWithUnknownETag(t *testing.T) {
	etag := `W/"878f43039ad0553d0d3122d8bc171b01"`
	as := makeGitLabAPIServer(t, testToken, "/api/v4/projects/testing/repo/repository/commits", "master", etag, mustReadFile(t, "testdata/gitlab_commit.json"))
	t.Cleanup(as.Close)
	g := NewGitLabPoller(as.Client(), as.URL, testToken)
	g.endpoint = as.URL

	polled, err := g.Poll("testing/repo", pollingv1alpha1.PollStatus{Ref: "master"})
	if err != nil {
		t.Fatal(err)
	}

	if polled.ETag != etag {
		t.Errorf("Poll() ETag got %s, want %s", polled.ETag, etag)
	}
	if polled.SHA != "ed899a2f4b50b4370feeea94676502b42383c746" {
		t.Errorf("Poll() SHA got %s, want %s", polled.SHA, "ed899a2f4b50b4370feeea94676502b42383c746")
	}
}

func TestGitLabWithKnownTag(t *testing.T) {
	etag := `W/"878f43039ad0553d0d3122d8bc171b01"`
	as := makeGitLabAPIServer(t, testToken, "/api/v4/projects/testing/repo/repository/commits", "master", etag, nil)
	t.Cleanup(as.Close)

	g := NewGitLabPoller(as.Client(), as.URL, testToken)
	g.endpoint = as.URL

	polled, err := g.Poll("testing/repo", pollingv1alpha1.PollStatus{Ref: "master", ETag: etag})
	if err != nil {
		t.Fatal(err)
	}
	if polled.ETag != etag {
		t.Fatalf("Poll() got %s, want %s", polled.ETag, etag)
	}
}

func TestGitLabWithNotFoundResponse(t *testing.T) {
	etag := `W/"878f43039ad0553d0d3122d8bc171b01"`
	as := makeGitLabAPIServer(t, testToken, "/api/v4/projects/testing/repo/repository/commits", "master", etag, nil)
	t.Cleanup(as.Close)
	g := NewGitLabPoller(as.Client(), as.URL, testToken)
	g.endpoint = as.URL

	_, err := g.Poll("testing/testing", pollingv1alpha1.PollStatus{Ref: "master", ETag: etag})
	if err.Error() != "server error: 404" {
		t.Fatal(err)
	}
}

// It's impossible to distinguish between unknown repo, and bad auth token, both
// respond with a 404.
func TestGitLabWithBadAuthentication(t *testing.T) {
	etag := `W/"878f43039ad0553d0d3122d8bc171b01"`
	as := makeGitLabAPIServer(t, testToken, "/api/v4/projects/testing/repo/repository/commits", "master", etag, nil)
	t.Cleanup(as.Close)
	g := NewGitLabPoller(as.Client(), as.URL, "anotherToken")
	g.endpoint = as.URL

	_, err := g.Poll("testing/repo", pollingv1alpha1.PollStatus{Ref: "master", ETag: etag})
	if err.Error() != "server error: 404" {
		t.Fatal(err)
	}
}

// With no auth-token, no auth header should be sent.
func TestGitLabWithNoAuthentication(t *testing.T) {
	etag := `W/"878f43039ad0553d0d3122d8bc171b01"`
	as := makeGitLabAPIServer(t, "", "/api/v4/projects/testing/repo/repository/commits", "master", etag, nil)
	t.Cleanup(as.Close)
	g := NewGitLabPoller(as.Client(), as.URL, "")
	g.endpoint = as.URL

	_, err := g.Poll("testing/repo", pollingv1alpha1.PollStatus{Ref: "master", ETag: etag})
	if err != nil {
		t.Fatal(err)
	}
}

// makeAPIServer is used during testing to create an HTTP server to return
// fixtures if the request matches.
func makeGitLabAPIServer(t *testing.T, authToken, wantPath, wantRef, etag string, response []byte) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Logf("got URL %#v, want %#v", r.URL.Path, wantPath)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if queryRef := r.URL.Query().Get("ref"); queryRef != wantRef {
			t.Errorf("got query ref %#v, want %#v", queryRef, wantRef)
		}
		if authToken != "" {
			if auth := r.Header.Get("Private-Token"); auth != authToken {
				t.Logf("got auth token %#v, want %#v", auth, authToken)
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}
		if auth := r.Header.Get("Private-Token"); auth != "" && authToken == "" {
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
