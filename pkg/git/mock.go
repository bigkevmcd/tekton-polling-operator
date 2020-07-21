package git

import (
	"strings"

	pollingv1 "github.com/bigkevmcd/tekton-polling-operator/pkg/apis/polling/v1alpha1"
)

var _ CommitPoller = (*MockPoller)(nil)

// NewMockPoller creates and returns a new mock Git poller.
func NewMockPoller() *MockPoller {
	return &MockPoller{
		responses: make(map[string]pollingv1.PollStatus),
		commits:   make(map[string]Commit),
	}
}

// MockPoller is a mock Git poller.
type MockPoller struct {
	pollError error
	responses map[string]pollingv1.PollStatus
	commits   map[string]Commit
}

// Poll is an implementation of the CommitPoller interface.
func (m *MockPoller) Poll(repo string, ps pollingv1.PollStatus) (pollingv1.PollStatus, Commit, error) {
	if m.pollError != nil {
		return pollingv1.PollStatus{}, nil, m.pollError
	}
	k := mockKey(repo, ps)
	return m.responses[k], m.commits[k], nil
}

// AddMockResponse sets up the response for a Poll call.
func (m *MockPoller) AddMockResponse(repo string, in pollingv1.PollStatus, c Commit, out pollingv1.PollStatus) {
	k := mockKey(repo, in)
	m.responses[k] = out
	m.commits[k] = c
}

// FailWithError configures the poller to return errors.
func (m *MockPoller) FailWithError(err error) {
	m.pollError = err
}

func mockKey(repo string, ps pollingv1.PollStatus) string {
	return strings.Join([]string{repo, ps.Ref, ps.SHA, ps.ETag}, ":")
}
