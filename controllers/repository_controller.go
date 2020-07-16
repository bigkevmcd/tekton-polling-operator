/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pollingv1alpha1 "github.com/bigkevmcd/tekton-polling-operator/api/v1alpha1"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/git"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/pipelines"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/secrets"
)

// commitPollerFactory creates a client for polling a specific endpoint.
type commitPollerFactory func(repo *pollingv1alpha1.Repository, endpoint, authToken string) git.CommitPoller

// RepositoryReconciler reconciles a Repository object
type RepositoryReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// The poller polls the endpoint for the repo.
	PollerFactory commitPollerFactory
	// The pipelineRunner executes the named pipeline with appropriate params.
	PipelineRunner pipelines.PipelineRunner
	SecretGetter   secrets.SecretGetter
}

// +kubebuilder:rbac:groups=polling.tekton.dev,resources=repositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=polling.tekton.dev,resources=repositories/status,verbs=get;update;patch

func (r *RepositoryReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	reqLogger := r.Log.WithValues("repository", req.NamespacedName)

	repo := &pollingv1alpha1.Repository{}
	err := r.Client.Get(ctx, req.NamespacedName, repo)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	repoName, endpoint, err := repoFromURL(repo.Spec.URL)
	if err != nil {
		reqLogger.Error(err, "Parsing the repo from the URL failed", "repoURL", repo.Spec.URL)
		return ctrl.Result{}, err
	}

	authToken, err := r.authTokenForRepo(ctx, reqLogger, req.Namespace, repo)
	if err != nil {
		return ctrl.Result{}, err
	}

	repo.Status.PollStatus.Ref = repo.Spec.Ref
	// TODO: handle pollerFactory returning nil/error
	newStatus, err := r.PollerFactory(repo, endpoint, authToken).Poll(repoName, repo.Status.PollStatus)
	if err != nil {
		repo.Status.LastError = err.Error()
		reqLogger.Error(err, "Repository poll failed")
		if err := r.Client.Status().Update(ctx, repo); err != nil {
			reqLogger.Error(err, "unable to update Repository status")
		}
		return ctrl.Result{}, err
	}

	repo.Status.LastError = ""
	changed := !newStatus.Equal(repo.Status.PollStatus)
	if repo.Status.LastError != "" {
		repo.Status.LastError = ""
		changed = true
	}
	if !changed {
		reqLogger.Info("Poll Status unchanged, requeueing next check", "frequency", repo.GetFrequency())
		return ctrl.Result{RequeueAfter: repo.GetFrequency()}, nil
	}

	reqLogger.Info("Poll Status changed", "status", newStatus)
	repo.Status.PollStatus = newStatus
	if err := r.Client.Status().Update(ctx, repo); err != nil {
		reqLogger.Error(err, "unable to update Repository status")
		return ctrl.Result{}, err
	}
	pr, err := r.PipelineRunner.Run(ctx, repo.Spec.Pipeline.Name, repo.Spec.URL, repo.Status.PollStatus.SHA)
	if err != nil {
		reqLogger.Error(err, "failed to create a PipelineRun", "pipelineName", repo.Spec.Pipeline.Name)
		return ctrl.Result{}, err
	}
	reqLogger.Info("PipelineRun created", "name", pr.ObjectMeta.Name)
	reqLogger.Info("Requeueing next check", "frequency", repo.GetFrequency())
	return ctrl.Result{RequeueAfter: repo.GetFrequency()}, nil
}

func (r *RepositoryReconciler) authTokenForRepo(ctx context.Context, logger logr.Logger, namespace string, repo *pollingv1alpha1.Repository) (string, error) {
	if repo.Spec.Auth == nil {
		return "", nil
	}
	key := "token"
	if repo.Spec.Auth.Key != "" {
		key = repo.Spec.Auth.Key
	}
	authToken, err := r.SecretGetter.SecretToken(ctx, types.NamespacedName{Name: repo.Spec.Auth.Name, Namespace: namespace}, key)
	if err != nil {
		logger.Error(err, "Getting the auth token failed", "name", repo.Spec.Auth.Name, "namespace", namespace, "key", key)
		return "", err
	}
	return authToken, nil
}

func (r *RepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pollingv1alpha1.Repository{}).
		Complete(r)
}

// TODO: create an HTTP client that has appropriate timeouts.
// TODO: pass the logger through so that we can log out errors from this and
// also the pipelinerun creator.
// MakeCommitPoller creates a new commit poller, bylooking at the type and URL
// and creating a client.
func MakeCommitPoller(repo *pollingv1alpha1.Repository, endpoint, authToken string) git.CommitPoller {
	switch repo.Spec.Type {
	case pollingv1alpha1.GitHub:
		if endpoint == "https://github.com" {
			endpoint = ""
		}
		return git.NewGitHubPoller(http.DefaultClient, endpoint, authToken)
	case pollingv1alpha1.GitLab:
		if endpoint == "https://gitlab.com" {
			endpoint = ""
		}
		return git.NewGitLabPoller(http.DefaultClient, endpoint, authToken)
	}
	return nil
}

func repoFromURL(s string) (string, string, error) {
	parsed, err := url.Parse(s)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse repo from URL %#v: %s", s, err)
	}

	endpoint := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	return strings.TrimPrefix(strings.TrimSuffix(parsed.Path, ".git"), "/"), endpoint, nil
}
