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
)

var log = logf.Log.WithName("controller_repository")

// Add creates a new Repository Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRepository{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("repository-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Repository
	err = c.Watch(&source.Kind{Type: &pollingv1alpha1.Repository{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileRepository implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRepository{}

// ReconcileRepository reconciles a Repository object
type ReconcileRepository struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
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
		return reconcile.Result{}, nil
	}
	repo.Status.PollStatus.Ref = repo.Spec.Ref
	newStatus, err := makeCommitPoller().Poll(repoName, repo.Status.PollStatus)
	if !newStatus.Equal(repo.Status.PollStatus) {
		reqLogger.Info("Poll Status changed", "status", newStatus)
	}

	if err != nil {
		repo.Status.LastError = err.Error()
		log.Error(err, "Repository poll failed")
		if err := r.client.Status().Update(ctx, repo); err != nil {
			log.Error(err, "unable to update Repository status")
		}
		return reconcile.Result{Requeue: true}, err
	}

	repo.Status.PollStatus = *newStatus
	if err := r.client.Status().Update(ctx, repo); err != nil {
		log.Error(err, "unable to update Repository status")
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{RequeueAfter: repo.GetFrequency()}, nil
}

func makeCommitPoller() git.CommitPoller {
	return git.NewGitHub(http.DefaultClient, "")
}

func repoFromURL(s string) (string, error) {
	parsed, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("failed to parse repo from URL %#v: %s", s, err)
	}
	return strings.TrimPrefix(strings.TrimSuffix(parsed.Path, ".git"), "/"), nil
}
