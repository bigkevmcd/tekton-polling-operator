package pipelines

import (
	"context"
	"fmt"
	"strings"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	resourcev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
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
func (c *ClientPipelineRunner) Run(ctx context.Context, pipelineName, ns, serviceAccountName string, params []pipelinev1.Param, res []pipelinev1.PipelineResourceBinding) (*pipelinev1.PipelineRun, error) {
	pr := c.makePipelineRun(pipelineName, serviceAccountName, ns, params, res)
	err := c.client.Create(ctx, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create a pipeline run for pipeline %s: %w", pipelineName, err)
	}
	return pr, nil
}

func (c *ClientPipelineRunner) makePipelineRun(pipelineName, serviceAccountName, ns string, params []pipelinev1.Param, res []pipelinev1.PipelineResourceBinding) *pipelinev1.PipelineRun {
	return &pipelinev1.PipelineRun{
		TypeMeta:   pipelineRunMeta,
		ObjectMeta: c.objectMeta(ns),
		Spec: pipelinev1.PipelineRunSpec{
			PipelineRef:        &pipelinev1.PipelineRef{Name: pipelineName},
			ServiceAccountName: serviceAccountName,
			Params:             params,
			Resources:          applyReplacements(res, params),
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

func applyReplacements(res []pipelinev1.PipelineResourceBinding, params []pipelinev1.Param) []pipelinev1.PipelineResourceBinding {
	updated := []pipelinev1.PipelineResourceBinding{}
	for _, r := range res {
		newParams := []resourcev1alpha1.ResourceParam{}
		for _, p := range r.ResourceSpec.Params {
			newParams = append(newParams, patchParam(params, p))
		}
		r.ResourceSpec.Params = newParams
		updated = append(updated, r)
	}
	return updated
}

// TODO: This is a grim hack - drop support for resources again.
func patchParam(params []pipelinev1.Param, param resourcev1alpha1.ResourceParam) resourcev1alpha1.ResourceParam {
	for _, p := range params {
		replace := fmt.Sprintf("$(params.%s)", p.Name)
		if p.Value.Type == "string" {
			param.Value = strings.ReplaceAll(param.Value, replace, p.Value.StringVal)
		}
	}
	return param
}
