package tekton

import (
	"encoding/json"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"

	"github.com/bigkevmcd/tekton-polling-operator/pkg/git"
)

type Resolver interface {
	Resolve(bindings []*triggersv1.EventListenerBinding, template triggersv1.EventListenerTemplate, commit git.Commit) ([]json.RawMessage, error)
}
