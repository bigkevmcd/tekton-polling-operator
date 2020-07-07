package secrets

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
)

var _ SecretGetter = (*MockSecret)(nil)

// NewMock returns a simple secret getter.
func NewMock() MockSecret {
	return MockSecret{}
}

// MockSecret implements the SecretGetter interface.
type MockSecret struct {
	secrets map[string]string
}

// SecretToken implements the SecretGetter interface.
func (k MockSecret) SecretToken(ctx context.Context, secretID types.NamespacedName) (string, error) {
	token, ok := k.secrets[key(secretID)]
	if !ok {
		return "", fmt.Errorf("mock not found")
	}
	return token, nil
}

// AddStubResponse is a mock method that sets up a token to be returned.
func (k MockSecret) AddStubResponse(authToken string, secretID types.NamespacedName, token string) {
	k.secrets[key(secretID)] = token
}

func key(n types.NamespacedName) string {
	return fmt.Sprintf("%s:%s", n.Name, n.Namespace)
}
