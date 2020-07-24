package tekton

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"

	"github.com/bigkevmcd/tekton-polling-operator/pkg/git"
)

type Resolver interface {
	Resolve(bindings []*triggersv1.EventListenerBinding, template triggersv1.EventListenerTemplate, commit git.Commit) ([]json.RawMessage, error)
}

type ResolverClient interface {
	GetTriggerBinding(name string, options metav1.GetOptions) (*triggersv1.TriggerBinding, error)
	GetTriggerTemplate(name string, options metav1.GetOptions) (*triggersv1.TriggerTemplate, error)
	GetClusterTriggerBinding(name string, options metav1.GetOptions) (*triggersv1.ClusterTriggerBinding, error)
}
