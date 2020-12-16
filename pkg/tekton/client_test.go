package tekton

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var _ ResolverClient = (*Client)(nil)

func TestGetTriggerBinding(t *testing.T) {
	binding := makeBinding()
	triggersClient := makeClient(t, binding)
	r := NewClient(binding.Namespace, triggersClient)

	loaded, err := r.GetTriggerBinding(binding.Name)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(binding, loaded); diff != "" {
		t.Fatalf("failed to load binding:\n%s", diff)
	}
}

func TestGetTriggerBinding_with_unknown_binding(t *testing.T) {
	triggersClient := makeClient(t)
	r := NewClient(testNS, triggersClient)

	_, err := r.GetTriggerBinding("unknown")
	if !matchError(t, "could not load TriggerBinding testing/unknown", err) {
		t.Fatal(err)
	}
}

func TestGetTriggerTemplate(t *testing.T) {
	template := makeTemplate(t)
	triggersClient := makeClient(t, template)
	r := NewClient(template.Namespace, triggersClient)

	loaded, err := r.GetTriggerTemplate(template.Name)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(template, loaded); diff != "" {
		t.Fatalf("failed to load template:\n%s", diff)
	}
}

func TestGetTriggerTemplate_with_unknown_template(t *testing.T) {
	triggersClient := makeClient(t)
	r := NewClient(testNS, triggersClient)

	_, err := r.GetTriggerTemplate("unknown")
	if !matchError(t, "could not load TriggerTemplate testing/unknown", err) {
		t.Fatal(err)
	}
}

func TestGetClusterTriggerBinding(t *testing.T) {
	binding := makeClusterBinding()
	triggersClient := makeClient(t, binding)

	r := NewClient(binding.Namespace, triggersClient)

	loaded, err := r.GetClusterTriggerBinding(binding.Name)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(binding, loaded); diff != "" {
		t.Fatalf("failed to load binding:\n%s", diff)
	}
}

func TestGetClusterTriggerBinding_with_unknown_binding(t *testing.T) {
	triggersClient := makeClient(t)
	r := NewClient(testNS, triggersClient)

	_, err := r.GetClusterTriggerBinding("unknown")
	if !matchError(t, "could not load ClusterTriggerBinding unknown", err) {
		t.Fatal(err)
	}
}

func matchError(t *testing.T, s string, e error) bool {
	t.Helper()
	if s == "" && e == nil {
		return true
	}
	if s != "" && e == nil {
		return false
	}
	match, err := regexp.MatchString(s, e.Error())
	if err != nil {
		t.Fatal(err)
	}
	return match
}
