package pipelines

import (
	"context"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// PipelineRunner executes a pipeline by name, passing in the SHA and RepURL as
// parameters.
type PipelineRunner interface {
	Run(ctx context.Context, pipelineName, ns string, params []pipelinev1.Param) (*pipelinev1.PipelineRun, error)
}
