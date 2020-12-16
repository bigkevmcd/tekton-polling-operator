package tekton

import (
	"context"
	"fmt"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client is an implementation of the ResolverClient that uses a basic client to
// fetch the resources.
type Client struct {
	ns     string
	client client.Client
}

// NewClient creates and returns a new Client.
func NewClient(ns string, c client.Client) *Client {
	return &Client{
		ns:     ns,
		client: c,
	}
}

// GetTriggerBinding is an implementation of the ResolverClient interface.
func (c *Client) GetTriggerBinding(name string) (*triggersv1.TriggerBinding, error) {
	v := &triggersv1.TriggerBinding{}
	err := c.client.Get(context.Background(), c.nsName(name), v)
	if err != nil {
		return nil, fmt.Errorf("could not load TriggerBinding %v: %w", c.nsName(name), err)
	}
	return v, nil
}

// GetTriggerTemplate is an implementation of the ResolverClient interface.
func (c *Client) GetTriggerTemplate(name string) (*triggersv1.TriggerTemplate, error) {
	v := &triggersv1.TriggerTemplate{}
	err := c.client.Get(context.Background(), c.nsName(name), v)
	if err != nil {
		return nil, fmt.Errorf("could not load TriggerTemplate %v: %w", c.nsName(name), err)
	}
	return v, nil
}

// GetTriggerTriggerBinding is an implementation of the ResolverClient interface.
func (c *Client) GetClusterTriggerBinding(name string) (*triggersv1.ClusterTriggerBinding, error) {
	v := &triggersv1.ClusterTriggerBinding{}
	err := c.client.Get(context.Background(), types.NamespacedName{Name: name}, v)
	if err != nil {
		return nil, fmt.Errorf("could not load ClusterTriggerBinding %s: %w", name, err)
	}
	return v, nil
}

func (c *Client) nsName(n string) types.NamespacedName {
	return types.NamespacedName{Name: n, Namespace: c.ns}
}
