package tekton

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type TriggersClient struct {
	client client.Client
}

func New(c client.Client) *TriggersClient {
	return &TriggersClient{client: c}
}

func (c *TriggersClient) GetTriggerBinding(name string, options metav1.GetOptions) (*triggersv1.TriggerBinding, error) {
	return nil, nil
}

func (c *TriggersClient) GetTriggerTemplate(name string, options metav1.GetOptions) (*triggersv1.TriggerTemplate, error) {
	return nil, nil
}

func (c *TriggersClient) GetClusterTriggerBinding(name string, options metav1.GetOptions) (*triggersv1.ClusterTriggerBinding, error) {
	return nil, nil
}

var _ ResolverClient = (*TriggersClient)(nil)

func TestGetTriggerBinding(t *testing.T) {
	tb := triggersv1.TriggerBinding{
		Spec: triggersv1.TriggerBindingSpec{
			Params: []triggersv1.Param{
				{Name: "test", Value: "value"},
			},
		},
	}
	g := New(fake.NewFakeClient(&tb))

	loaded, err := g.GetTriggerBinding("my-test-tb", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	want := &triggersv1.TriggerBinding{}
	if diff := cmp.Diff(want, loaded); diff != "" {
		t.Fatalf("GetTriggerBinding() failed:\n%s", diff)
	}

}
func TestGetTriggerTemplate(t *testing.T) {
	_ = New(fake.NewFakeClient())
}
func TestGetClusterTriggerBinding(t *testing.T) {
	_ = New(fake.NewFakeClient())
}
