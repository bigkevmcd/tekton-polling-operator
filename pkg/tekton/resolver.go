package tekton

import (
	"encoding/json"
	"fmt"
	"net/http"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"github.com/tektoncd/triggers/pkg/template"

	"github.com/bigkevmcd/tekton-polling-operator/pkg/git"
)

// New creates a new ResourceResolver.
type ResourceResolver struct {
	clientFactory func(string) ResolverClient
}

// New creates and returns a New ResourceResolver.
func New(c triggersclientset.Interface) *ResourceResolver {
	return &ResourceResolver{clientFactory: clientFactory(c)}
}

func (r ResourceResolver) Resolve(ns string, bindings []*triggersv1.EventListenerBinding, tt triggersv1.EventListenerTemplate, commit git.Commit) ([]json.RawMessage, error) {
	trigger := triggersv1.Trigger{
		Spec: triggersv1.TriggerSpec{
			Bindings: bindings,
			Template: tt,
		},
	}
	client := r.clientFactory(ns)
	rt, err := template.ResolveTrigger(trigger,
		client.GetTriggerBinding,
		client.GetClusterTriggerBinding,
		client.GetTriggerTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve trigger: %w", err)
	}

	payload, err := json.Marshal(commit)
	if err != nil {
		return nil, err
	}
	params, err := template.ResolveParams(rt, payload, http.Header{}, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	resources := template.ResolveResources(rt.TriggerTemplate, params)
	return resources, nil
}

func clientFactory(c triggersclientset.Interface) func(string) ResolverClient {
	return func(ns string) ResolverClient {
		return NewVersionedClient(ns, c)
	}
}
