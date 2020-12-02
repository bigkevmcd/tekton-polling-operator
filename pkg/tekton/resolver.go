package tekton

import (
	"encoding/json"
	"net/http"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"github.com/tektoncd/triggers/pkg/template"

	"github.com/bigkevmcd/tekton-polling-operator/pkg/git"
)

type ResourceResolver struct {
	client triggersclientset.Interface
}

func New(c triggersclientset.Interface) *ResourceResolver {
	return &ResourceResolver{client: c}
}

func (r ResourceResolver) Resolve(ns string, bindings []*triggersv1.EventListenerBinding, tt triggersv1.EventListenerTemplate, commit git.Commit) ([]json.RawMessage, error) {
	trigger := triggersv1.EventListenerTrigger{
		Bindings: bindings,
		Template: tt,
	}

	rt, err := template.ResolveTrigger(trigger,
		r.client.TriggersV1alpha1().TriggerBindings(ns).Get,
		r.client.TriggersV1alpha1().ClusterTriggerBindings().Get,
		r.client.TriggersV1alpha1().TriggerTemplates(ns).Get)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(commit)
	if err != nil {
		return nil, err
	}
	params, err := template.ResolveParams(rt, payload, http.Header{})
	if err != nil {
		return nil, err
	}
	resources := template.ResolveResources(rt.TriggerTemplate, params)
	return resources, nil
}
