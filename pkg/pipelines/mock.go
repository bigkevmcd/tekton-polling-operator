package pipelines

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

var _ PipelineRunner = (*MockRunner)(nil)

// NewMockRunner creates and returns a new mock PipelineRunner.
func NewMockRunner(t *testing.T) *MockRunner {
	return &MockRunner{runs: make(map[string]run), t: t}
}

// MockRunner is a mock pipeline runner that returns fixed responses to runs.
type MockRunner struct {
	t        *testing.T
	runs     map[string]run
	runError error
}

type run struct {
	params []pipelinev1.Param
}

// Run is an implementation of the PipelineRunner interface.
func (m *MockRunner) Run(ctx context.Context, pipelineName, ns string, params []pipelinev1.Param) (*pipelinev1.PipelineRun, error) {
	if m.runError != nil {
		return nil, m.runError
	}
	m.runs[mockKey(ns, pipelineName)] = run{params: params}
	return &pipelinev1.PipelineRun{}, nil
}

// AssertPipelineRun ensures that the pipeline run was triggered.
func (m *MockRunner) AssertPipelineRun(pipelineName, ns string, wantParams []pipelinev1.Param) {
	m.t.Helper()
	run, ok := m.runs[mockKey(ns, pipelineName)]
	if !ok {
		m.t.Fatalf("no pipeline run for %s/%s", ns, pipelineName)
	}
	if diff := cmp.Diff(wantParams, run.params); diff != "" {
		m.t.Fatalf("incorrect params for pipelinerun:\n%s", diff)
	}
}

// AssertNoPipelineRuns fails if there were any pipelines executed.
func (m *MockRunner) AssertNoPipelineRuns() {
	m.t.Helper()
	if len(m.runs) != 0 {
		m.t.Fatalf("pipelines were executed: %#v\n", m.runs)
	}
}

// FailWithError configures the poller to return errors.
func (m *MockRunner) FailWithError(err error) {
	m.runError = err
}

func mockKey(ns, pipelineName string) string {
	return strings.Join([]string{ns, pipelineName}, ":")
}
