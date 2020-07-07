package git

import (
	"strings"

	pollingv1 "github.com/bigkevmcd/tekton-polling-operator/pkg/apis/polling/v1alpha1"
)

// NewStubPoller creates and returns a new mock Git poller.
func NewStubPoller() *Stub {
	return &Stub{
		responses: make(map[string]*pollingv1.PollStatus),
	}
}

// Stub is a stub Git poller.
type Stub struct {
	responses map[string]*pollingv1.PollStatus
}

// Poll is an implementation of the CommitPoller interface.
func (m *Stub) Poll(repo string, ps pollingv1.PollStatus) (*pollingv1.PollStatus, error) {
	return m.responses[stubKey(repo, ps)], nil
}

// AddStubResponse sets up the response for a Poll call.
func (m *Stub) AddStubResponse(repo string, in pollingv1.PollStatus, out *pollingv1.PollStatus) {
	m.responses[stubKey(repo, in)] = out
}

func stubKey(repo string, ps pollingv1.PollStatus) string {
	return strings.Join([]string{repo, ps.Ref, ps.SHA, ps.ETag}, ":")
}
