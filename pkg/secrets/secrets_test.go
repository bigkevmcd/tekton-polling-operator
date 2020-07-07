package secrets

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ SecretGetter = (*KubeSecretGetter)(nil)

var testID = types.NamespacedName{Name: "test-secret", Namespace: "test-ns"}

func TestSecretToken(t *testing.T) {
	g := New(fake.NewFakeClient(createSecret(testID, "secret-token")))

	secret, err := g.SecretToken(context.TODO(), testID)
	if err != nil {
		t.Fatal(err)
	}

	if secret != "secret-token" {
		t.Fatalf("got %s, want secret-token", secret)
	}
}

func TestSecretTokenWithMissingSecret(t *testing.T) {
	g := New(fake.NewFakeClient())

	_, err := g.SecretToken(context.TODO(), testID)
	if err.Error() != `error getting secret test-ns/test-secret: secrets "test-secret" not found` {
		t.Fatal(err)
	}
}

func createSecret(id types.NamespacedName, token string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      id.Name,
			Namespace: id.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"token": []byte(token),
		},
	}
}
