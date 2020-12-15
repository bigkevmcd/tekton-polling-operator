package tekton

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var _ ResolverClient = (*VersionedClient)(nil)

func TestGetTriggerBinding(t *testing.T) {
	binding := makeBinding()
	triggersClient := makeClient(binding)

	r := NewVersionedClient(binding.Namespace, triggersClient)

	loaded, err := r.GetTriggerBinding(binding.Name)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(binding, loaded); diff != "" {
		t.Fatalf("failed to load binding:\n%s", diff)
	}
}

func TestGetTriggerTemplate(t *testing.T) {
	template := makeTemplate(t)
	triggersClient := makeClient(template)

	r := NewVersionedClient(template.Namespace, triggersClient)

	loaded, err := r.GetTriggerTemplate(template.Name)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(template, loaded); diff != "" {
		t.Fatalf("failed to load template:\n%s", diff)
	}
}

func TestGetClusterTriggerBinding(t *testing.T) {
	binding := makeClusterBinding()
	triggersClient := makeClient(binding)

	r := NewVersionedClient(binding.Namespace, triggersClient)

	loaded, err := r.GetClusterTriggerBinding(binding.Name)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(binding, loaded); diff != "" {
		t.Fatalf("failed to load binding:\n%s", diff)
	}
}
