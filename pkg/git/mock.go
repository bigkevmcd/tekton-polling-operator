package git

import (
	"strings"

	pollingv1alpha1 "github.com/bigkevmcd/tekton-polling-operator/api/v1alpha1"
)

// NewMockPoller creates and returns a new mock Git poller.
func NewMockPoller() *MockPoller {
	return &MockPoller{
		responses: make(map[string]pollingv1alpha1.PollStatus),
	}
}

// MockPoller is a mock Git poller.
type MockPoller struct {
	pollError error
	responses map[string]pollingv1alpha1.PollStatus
}

// Poll is an implementation of the CommitPoller interface.
func (m *MockPoller) Poll(repo string, ps pollingv1alpha1.PollStatus) (pollingv1alpha1.PollStatus, error) {
	if m.pollError != nil {
		return pollingv1alpha1.PollStatus{}, m.pollError
	}
	return m.responses[mockKey(repo, ps)], nil
}

// AddMockResponse sets up the response for a Poll call.
func (m *MockPoller) AddMockResponse(repo string, in pollingv1alpha1.PollStatus, out pollingv1alpha1.PollStatus) {
	m.responses[mockKey(repo, in)] = out
}

// FailWithError configures the poller to return errors.
func (m *MockPoller) FailWithError(err error) {
	m.pollError = err
}

func mockKey(repo string, ps pollingv1alpha1.PollStatus) string {
	return strings.Join([]string{repo, ps.Ref, ps.SHA, ps.ETag}, ":")
}
