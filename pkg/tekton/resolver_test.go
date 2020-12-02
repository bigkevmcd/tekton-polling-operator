package tekton

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	fake "github.com/tektoncd/triggers/pkg/client/clientset/versioned/fake"
	"k8s.io/apimachinery/pkg/runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ Resolver = (*ResourceResolver)(nil)

const testNS = "testing"

func TestResolveWithKnownResources(t *testing.T) {
	binding := makeBinding()
	template := makeTemplate(t)
	triggersClient := makeClient(t, context.Background(),
		binding, template)

	commit := map[string]interface{}{
		"id": "1f18b9248b11b31a4dc5d36af4f8acadd5fbb76e",
	}
	templateBinding := triggersv1.EventListenerTemplate{Name: template.Name}
	r := New(triggersClient)

	resolved, err := r.Resolve(testNS, makeEventListenerBindings(binding), templateBinding, commit)
	if err != nil {
		t.Fatal(err)
	}
	unmarshaled := make([]map[string]interface{}, len(resolved))
	for i, v := range resolved {
		u := map[string]interface{}{}
		err := json.Unmarshal(v, &u)
		if err != nil {
			t.Fatal(err)
		}
		unmarshaled[i] = u
	}

	want := []map[string]interface{}{
		{
			"kind": "PipelineRun", "apiVersion": "tekton.dev/v1beta1", "metadata": map[string]interface{}{
				"name":              "test-pipeline-run",
				"namespace":         "testing",
				"creationTimestamp": nil},
			"spec": map[string]interface{}{
				"pipelineRef": map[string]interface{}{
					"name": "test-pipeline"},
			},
			"status": map[string]interface{}{},
		},
	}
	if diff := cmp.Diff(want, unmarshaled); diff != "" {
		t.Fatalf("resolved resources:\n%s", diff)
	}
}

func TestResolveWithMissingResources(t *testing.T) {
	t.Skip()
}

func makeEventListenerBindings(b *triggersv1.TriggerBinding) []*triggersv1.EventListenerBinding {
	return []*triggersv1.EventListenerBinding{
		{
			Name: b.Name,
			Kind: "TriggerBinding",
		},
	}
}

func makeTemplate(t *testing.T) *triggersv1.TriggerTemplate {
	typeMeta := metav1.TypeMeta{
		APIVersion: "triggers.tekton.dev/v1alpha1",
		Kind:       "TriggerTemplate",
	}
	pipelineRunMeta := metav1.TypeMeta{
		APIVersion: "tekton.dev/v1beta1",
		Kind:       "PipelineRun",
	}
	return &triggersv1.TriggerTemplate{
		TypeMeta: typeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-template",
			Namespace: testNS,
		},
		Spec: triggersv1.TriggerTemplateSpec{
			Params: []triggersv1.ParamSpec{
				{Name: "gitrevision"},
			},
			ResourceTemplates: []triggersv1.TriggerResourceTemplate{
				{
					RawExtension: runtime.RawExtension{
						Raw: mustMarshal(t, pipelinev1.PipelineRun{
							TypeMeta: pipelineRunMeta,
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-pipeline-run",
								Namespace: testNS,
							},
							Spec: pipelinev1.PipelineRunSpec{
								PipelineRef: &pipelinev1.PipelineRef{
									Name: "test-pipeline",
								},
							},
						}),
					},
				},
			},
		},
	}
}

func makeBinding() *triggersv1.TriggerBinding {
	typeMeta := metav1.TypeMeta{
		APIVersion: "triggers.tekton.dev/v1alpha1",
		Kind:       "TriggerBinding",
	}
	return &triggersv1.TriggerBinding{
		TypeMeta: typeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-binding",
			Namespace: testNS,
		},
		Spec: triggersv1.TriggerBindingSpec{
			Params: []triggersv1.Param{
				{Name: "gitrevision", Value: "$(body.id)"},
			},
		},
	}
}

func makeClient(t *testing.T, ctx context.Context, objs ...runtime.Object) triggersclientset.Interface {
	return fake.NewSimpleClientset(objs...)
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v) // createDevCDPipelineRun(saName))
	if err != nil {
		t.Fatal(err)
	}
	return b
}
