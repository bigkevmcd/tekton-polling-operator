package pipelines

import (
	"context"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// PipelineRunner executes a pipeline by name, passing in the SHA and RepURL as
// parameters.
type PipelineRunner interface {
	Run(ctx context.Context, repoURL, sha string) (*pipelinev1.PipelineRun, error)
}
