package pipelines

import (
	"context"
	"testing"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	resourcev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/google/go-cmp/cmp"
)

const (
	testPipelineName = "test-pipeline"
	testPipelineRun  = "test-pipeline-run"
	testRepoURL      = "https://github.com/example/example.git"
	testSHA          = "35576600886452a3f0f2e9d459924865f4007614"
	testNamespace    = "test-namespace"
)

var _ PipelineRunner = (*ClientPipelineRunner)(nil)

func TestRunPipelineCreatesPipelineRun(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(pipelinev1.SchemeGroupVersion, &pipelinev1.PipelineRun{})
	cl := fake.NewFakeClient()
	r := NewRunner(cl)
	r.objectMeta = func(ns string) metav1.ObjectMeta {
		return metav1.ObjectMeta{
			Name:      testPipelineRun,
			Namespace: ns,
		}
	}

	params := []pipelinev1.Param{
		{Name: "test", Value: *pipelinev1.NewArrayOrString("value")},
	}
	resources := []pipelinev1.PipelineResourceBinding{
		{
			Name: "testing",
			ResourceSpec: &resourcev1alpha1.PipelineResourceSpec{
				Params: []resourcev1alpha1.ResourceParam{
					{
						Name:  "testing",
						Value: "$(params.test)",
					},
				},
			},
		},
	}
	_, err := r.Run(context.Background(), testPipelineName, testNamespace, params, resources)

	pr := &pipelinev1.PipelineRun{}
	err = cl.Get(context.Background(), types.NamespacedName{
		Namespace: testNamespace, Name: testPipelineRun,
	}, pr)
	if err != nil {
		t.Fatalf("get pipelinerun: %s", err)
	}

	want := &pipelinev1.PipelineRun{
		TypeMeta: pipelineRunMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:            testPipelineRun,
			Namespace:       testNamespace,
			ResourceVersion: "1",
		},
		Spec: pipelinev1.PipelineRunSpec{
			Params:      params,
			PipelineRef: &pipelinev1.PipelineRef{Name: testPipelineName},
			Resources: []pipelinev1.PipelineResourceBinding{
				{
					Name: "testing",
					ResourceSpec: &resourcev1alpha1.PipelineResourceSpec{
						Params: []resourcev1alpha1.ResourceParam{
							{
								Name:  "testing",
								Value: "value",
							},
						},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(pr, want); diff != "" {
		t.Fatalf("got an incorrect PipelineRun back:\n%s", diff)
	}
}

func TestApplyReplacements(t *testing.T) {
	resources := []pipelinev1.PipelineResourceBinding{
		{
			Name: "testing",
			ResourceSpec: &resourcev1alpha1.PipelineResourceSpec{
				Params: []resourcev1alpha1.ResourceParam{
					{
						Name:  "testing",
						Value: "$(params.test)",
					},
				},
			},
		},
	}
	params := []pipelinev1.Param{
		{Name: "test", Value: *pipelinev1.NewArrayOrString("value")},
	}

	replacements := applyReplacements(resources, params)

	want := []pipelinev1.PipelineResourceBinding{
		{
			Name: "testing",
			ResourceSpec: &resourcev1alpha1.PipelineResourceSpec{
				Params: []resourcev1alpha1.ResourceParam{
					{
						Name:  "testing",
						Value: "value",
					},
				},
			},
		},
	}

	if diff := cmp.Diff(want, replacements); diff != "" {
		t.Fatalf("applyReplacements:\n%s", diff)
	}
}
