package pipelines

import (
	"context"
	"strings"
	"testing"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// NewMockRunner creates and returns a new mock PipelineRunner.
func NewMockRunner(t *testing.T) *MockRunner {
	return &MockRunner{runs: make(map[string]string), t: t}
}

// MockRunner is a mock pipeline runner that returns fixed responses to runs.
type MockRunner struct {
	t        *testing.T
	runs     map[string]string
	runError error
}

// Run is an implementation of the PipelineRunner interface.
func (m *MockRunner) Run(ctx context.Context, pipelineName, repoURL, sha string) (*pipelinev1.PipelineRun, error) {
	if m.runError != nil {
		return nil, m.runError
	}
	m.runs[mockKey(pipelineName, repoURL)] = sha
	return &pipelinev1.PipelineRun{}, nil
}

// AssertPipelineRun ensures that the pipeline run was triggered.
func (m *MockRunner) AssertPipelineRun(pipelineName, repoURL, wantSHA string) {
	sha, ok := m.runs[mockKey(pipelineName, repoURL)]
	if !ok {
		m.t.Fatalf("no pipeline run for %s / %s", pipelineName, repoURL)
	}
	if sha != wantSHA {
		m.t.Fatalf("incorrect sha for pipeline run, got %#v, want %#v", sha, wantSHA)
	}
}

// RefutePipelineRun ensures that the pipeline run was triggered.
func (m *MockRunner) RefutePipelineRun(pipelineName, repoURL, wantSHA string) {
	sha := m.runs[mockKey(pipelineName, repoURL)]
	if sha == wantSHA {
		m.t.Fatalf("pipeline run with SHA %#v was run", wantSHA)
	}
}

// FailWithError configures the poller to return errors.
func (m *MockRunner) FailWithError(err error) {
	m.runError = err
}

func mockKey(pipelineName, repoURL string) string {
	return strings.Join([]string{pipelineName, repoURL}, ":")
}
