package repository

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	pollingv1alpha1 "github.com/bigkevmcd/tekton-polling-operator/pkg/apis/polling/v1alpha1"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/git"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/pipelines"
)

var log = logf.Log.WithName("controller_repository")

// Add creates a new Repository Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRepository{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),

		poller:         makeCommitPoller(),
		pipelineRunner: pipelines.NewRunner(mgr.GetClient()),
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("repository-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &pollingv1alpha1.Repository{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	return nil
}

// ReconcileRepository reconciles a Repository object.
type ReconcileRepository struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	// The poller polls the endpoint for the repo.
	poller git.CommitPoller
	// The pipelineRunner executes the named pipeline with appropriate params.
	pipelineRunner pipelines.PipelineRunner
}

// Reconcile reads that state of the cluster for a Repository object and makes changes based on the state read
// and what is in the Repository.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRepository) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Repository")
	ctx := context.Background()

	repo := &pollingv1alpha1.Repository{}
	err := r.client.Get(ctx, request.NamespacedName, repo)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	repoName, err := repoFromURL(repo.Spec.URL)
	if err != nil {
		log.Error(err, "Parsing the repo from the URL failed", "repoURL", repo.Spec.URL)
		return reconcile.Result{}, err
	}
	repo.Status.PollStatus.Ref = repo.Spec.Ref
	newStatus, err := r.poller.Poll(repoName, repo.Status.PollStatus)
	if err != nil {
		repo.Status.LastError = err.Error()
		log.Error(err, "Repository poll failed")
		if err := r.client.Status().Update(ctx, repo); err != nil {
			log.Error(err, "unable to update Repository status")
		}
		return reconcile.Result{Requeue: true}, err
	}

	repo.Status.LastError = ""
	changed := !newStatus.Equal(repo.Status.PollStatus)

	if !changed {
		reqLogger.Info("Poll Status unchanged, requeueing next check", "frequency", repo.GetFrequency())
		return reconcile.Result{RequeueAfter: repo.GetFrequency()}, nil
	}

	reqLogger.Info("Poll Status changed", "status", newStatus)
	repo.Status.PollStatus = *newStatus
	if err := r.client.Status().Update(ctx, repo); err != nil {
		log.Error(err, "unable to update Repository status")
		return reconcile.Result{Requeue: true}, err
	}
	pr, err := r.pipelineRunner.Run(ctx, repo.Spec.Pipeline.Name, repo.Spec.URL, repo.Status.PollStatus.SHA)
	if err != nil {
		log.Error(err, "failed to create a PipelineRun", "pipelineName", repo.Spec.Pipeline.Name)
		return reconcile.Result{Requeue: true}, err
	}
	reqLogger.Info("PipelineRun created", "name", pr.ObjectMeta.Name)
	reqLogger.Info("Requeueing next check", "frequency", repo.GetFrequency())
	return reconcile.Result{RequeueAfter: repo.GetFrequency()}, nil
}

// TODO: create an HTTP client that has appropriate timeouts.
// TODO: this needs to be moved to a factory so we can authenticate each poll
// when it's reconciled.
func makeCommitPoller() git.CommitPoller {
	return git.NewGitHubPoller(http.DefaultClient, "")
}

func repoFromURL(s string) (string, error) {
	parsed, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("failed to parse repo from URL %#v: %s", s, err)
	}
	return strings.TrimPrefix(strings.TrimSuffix(parsed.Path, ".git"), "/"), nil
}
