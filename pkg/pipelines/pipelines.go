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
// TODO: This should replaced by TriggerTemplates/TriggerBindings.
func (c *ClientPipelineRunner) Run(ctx context.Context, pipelineName, ns string, params []pipelinev1.Param) (*pipelinev1.PipelineRun, error) {
	pr := c.makePipelineRun(pipelineName, ns, params)
	err := c.client.Create(ctx, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create a pipeline run for pipeline %s: %w", pipelineName, err)
	}
	return pr, nil
}

func (c *ClientPipelineRunner) makePipelineRun(pipelineName, ns string, params []pipelinev1.Param) *pipelinev1.PipelineRun {
	return &pipelinev1.PipelineRun{
		TypeMeta:   pipelineRunMeta,
		ObjectMeta: c.objectMeta(ns),
		Spec: pipelinev1.PipelineRunSpec{
			PipelineRef: &pipelinev1.PipelineRef{Name: pipelineName},
			Params:      params,
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
