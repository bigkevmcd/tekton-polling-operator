package secrets

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KubeSecretGetter is an implementation of SecretGetter.
type KubeSecretGetter struct {
	kubeClient client.Client
}

// New creates and returns a KubeSecretGetter that looks up secrets in k8s.
func New(c client.Client) *KubeSecretGetter {
	return &KubeSecretGetter{
		kubeClient: c,
	}
}

// SecretToken looks for a namespaced secret, and returns the 'token' key from
// it, or an error if not found.
func (k KubeSecretGetter) SecretToken(ctx context.Context, id types.NamespacedName) (string, error) {
	loaded := &corev1.Secret{}
	err := k.kubeClient.Get(context.TODO(), id, loaded)
	if err != nil {
		return "", fmt.Errorf("error getting secret %s/%s: %w", id.Namespace, id.Name, err)
	}
	token, ok := loaded.Data["token"]
	if !ok {
		return "", fmt.Errorf("secret invalid, no 'token' key in %s/%s", id.Namespace, id.Name)
	}
	return string(token), nil
}
