package pipelines

import (
	"context"
	"fmt"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const pipelineRunNames = "polled-pipelinerun-"

var pipelineRunMeta = metav1.TypeMeta{
	APIVersion: "tekton.dev/v1beta1",
	Kind:       "PipelineRun",
}

// NewRunner creates a new PipelineRunner that creates PipelineRuns with the
// provided client.
func NewRunner(c client.Client) *ClientPipelineRunner {
	return &ClientPipelineRunner{client: c, objectMeta: objectMetaCreator}
}

// ClientPipelineRunner uses a split client to run pipelines.
type ClientPipelineRunner struct {
	client     client.Client
	objectMeta func(string) metav1.ObjectMeta
}

// Run is an implementation of the PipelineRunner interface.
func (c *ClientPipelineRunner) Run(ctx context.Context, pipelineName, ns, repoURL, sha string) (*pipelinev1.PipelineRun, error) {
	pr := c.makePipelineRun(pipelineName, ns, repoURL, sha)
	err := c.client.Create(ctx, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create a pipeline run for pipeline %s: %w", pipelineName, err)
	}
	return pr, nil
}

func (c *ClientPipelineRunner) makePipelineRun(pipelineName, ns, repoURL, sha string) *pipelinev1.PipelineRun {
	return &pipelinev1.PipelineRun{
		TypeMeta:   pipelineRunMeta,
		ObjectMeta: c.objectMeta(ns),
		Spec: pipelinev1.PipelineRunSpec{
			PipelineRef: &pipelinev1.PipelineRef{Name: pipelineName},
			Params: []pipelinev1.Param{
				{Name: "sha", Value: pipelinev1.NewArrayOrString(sha)},
				{Name: "repoURL", Value: pipelinev1.NewArrayOrString(repoURL)},
			},
		},
	}
}

// This is here because the controller-runtime fake client doesn't generate
// names...
func objectMetaCreator(ns string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		GenerateName: pipelineRunNames,
		Namespace:    ns,
	}
}
