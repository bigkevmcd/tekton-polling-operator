package secrets

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
)

// SecretGetter takes a namespaced name and finds a secret with that name, or
// returns an error.
type SecretGetter interface {
	SecretToken(ctx context.Context, id types.NamespacedName, key string) (string, error)
}
