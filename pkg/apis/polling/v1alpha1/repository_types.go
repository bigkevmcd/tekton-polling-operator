package v1alpha1

import (
	"time"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RepoType defines the protocol to use to talk to the upstream server.
// +kubebuilder:validation:Enum=github;gitlab
type RepoType string

const (
	GitHub RepoType = "github"
	GitLab RepoType = "gitlab"
)

// RepositorySpec defines a repository to poll.
type RepositorySpec struct {
	URL       string           `json:"url"`
	Ref       string           `json:"ref,omitempty"`
	Auth      *AuthSecret      `json:"auth,omitempty"`
	Type      RepoType         `json:"type,omitempty"`
	Frequency *metav1.Duration `json:"frequency,omitempty"`
	Pipeline  PipelineRef      `json:"pipelineRef"`
}

// PipelineRef links to the Pipeline to execute.
type PipelineRef struct {
	Name      string  `json:"name"`
	Namespace string  `json:"namespace,omitempty"`
	Params    []Param `json:"params,omitempty"`

	Bindings []*triggersv1.EventListenerBinding `json:"bindings"`
	Template triggersv1.EventListenerTemplate   `json:"template"`
}

type Param struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

// AuthSecret references a secret for authenticating the request.
type AuthSecret struct {
	corev1.SecretReference `json:"secretRef,omitempty"`
	Key                    string `json:"key,omitempty"`
}

// RepositoryStatus defines the observed state of Repository
type RepositoryStatus struct {
	LastError          string `json:"lastError,omitempty"`
	PollStatus         `json:"pollStatus,omitempty"`
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// PollStatus represents the last polled state of the repo.
type PollStatus struct {
	Ref  string `json:"ref"`
	SHA  string `json:"sha"`
	ETag string `json:"etag"`
}

// Equal returns true if two PollStatus values match.
func (p PollStatus) Equal(o PollStatus) bool {
	return (p.Ref == o.Ref) && (p.SHA == o.SHA) && (p.ETag == o.ETag)
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Repository is the Schema for the repositories API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=repositories,scope=Namespaced
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec,omitempty"`
	Status RepositoryStatus `json:"status,omitempty"`
}

// GetFrequency returns the configured delay between polls.
func (r *Repository) GetFrequency() time.Duration {
	if r.Spec.Frequency != nil {
		return r.Spec.Frequency.Duration
	}
	return time.Second * 30
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RepositoryList contains a list of Repository
type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repository `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Repository{}, &RepositoryList{})
}
