package repository

import (
	"context"
	"fmt"
	syslog "log"
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

	pollingv1 "github.com/bigkevmcd/tekton-polling-operator/pkg/apis/polling/v1alpha1"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/git"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/secrets"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/tekton"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
)

// Add creates a new Repository Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

type commitPollerFactory func(repo *pollingv1.Repository, endpoint, authToken string) git.CommitPoller

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRepository{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		pollerFactory: func(repo *pollingv1.Repository, endpoint, token string) git.CommitPoller {
			return makeCommitPoller(repo, endpoint, token)
		},
		secretGetter: secrets.New(mgr.GetClient()),
		log:          logf.Log.WithName("controller_repository"),
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("repository-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &pollingv1.Repository{}}, &handler.EnqueueRequestForObject{})
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
	pollerFactory commitPollerFactory
	secretGetter  secrets.SecretGetter
	log           logr.Logger
}

// Reconcile reads that state of the cluster for a Repository object and makes changes based on the state read
// and what is in the Repository.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRepository) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	reqLogger := r.log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling Repository")
	ctx := context.Background()

	repo := &pollingv1.Repository{}
	err := r.client.Get(ctx, req.NamespacedName, repo)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	repoName, endpoint, err := repoFromURL(repo.Spec.URL)
	if err != nil {
		reqLogger.Error(err, "Parsing the repo from the URL failed", "repoURL", repo.Spec.URL)
		return reconcile.Result{}, err
	}

	authToken, err := r.authTokenForRepo(ctx, reqLogger, req.Namespace, repo)
	if err != nil {
		return reconcile.Result{}, err
	}

	repo.Status.PollStatus.Ref = repo.Spec.Ref
	// TODO: handle pollerFactory returning nil/error
	newStatus, commit, err := r.pollerFactory(repo, endpoint, authToken).Poll(repoName, repo.Status.PollStatus)
	if err != nil {
		repo.Status.LastError = err.Error()
		reqLogger.Error(err, "Repository poll failed")
		if err := r.client.Status().Update(ctx, repo); err != nil {
			reqLogger.Error(err, "unable to update Repository status")
		}
		return reconcile.Result{}, err
	}

	repo.Status.LastError = ""
	changed := !newStatus.Equal(repo.Status.PollStatus)
	if repo.Status.LastError != "" {
		repo.Status.LastError = ""
		changed = true
	}
	if !changed {
		reqLogger.Info("Poll Status unchanged, requeueing next check", "frequency", repo.GetFrequency())
		return reconcile.Result{RequeueAfter: repo.GetFrequency()}, nil
	}

	reqLogger.Info("Poll Status changed", "status", newStatus)
	repo.Status.PollStatus = newStatus
	if err := r.client.Status().Update(ctx, repo); err != nil {
		reqLogger.Error(err, "unable to update Repository status")
		return reconcile.Result{}, err
	}

	resolver := tekton.New(r.client)
	resources, err := resolver.Resolve(req.Namespace, repo.Spec.Pipeline.Bindings, repo.Spec.Pipeline.Template, commit)
	if err != nil {
		reqLogger.Error(err, "failed to create resolve resources")
		return reconcile.Result{}, err
	}
	syslog.Printf("KEVIN!!!!! %s\n", resources)
	// reqLogger.Info("PipelineRun created", "name", pr.ObjectMeta.Name)
	reqLogger.Info("Requeueing next check", "frequency", repo.GetFrequency())
	return reconcile.Result{RequeueAfter: repo.GetFrequency()}, nil
}

func (r *ReconcileRepository) authTokenForRepo(ctx context.Context, logger logr.Logger, namespace string, repo *pollingv1.Repository) (string, error) {
	if repo.Spec.SecretRef == nil {
		return "", nil
	}
	key := "token"
	if repo.Spec.SecretRef.SecretKey != "" {
		key = repo.Spec.SecretRef.SecretKey
	}
	authToken, err := r.secretGetter.SecretToken(ctx, types.NamespacedName{Name: repo.Spec.SecretRef.SecretName, Namespace: namespace}, key)
	if err != nil {
		logger.Error(err, "Getting the auth token failed", "name", repo.Spec.SecretRef.SecretName, "namespace", namespace, "key", key)
		return "", err
	}
	return authToken, nil
}

// TODO: create an HTTP client that has appropriate timeouts.
// TODO: pass the logger through so that we can log out errors from this and
// also the pipelinerun creator.
func makeCommitPoller(repo *pollingv1.Repository, endpoint, authToken string) git.CommitPoller {
	switch repo.Spec.Type {
	case pollingv1.GitHub:
		return git.NewGitHubPoller(http.DefaultClient, endpoint, authToken)
	case pollingv1.GitLab:
		return git.NewGitLabPoller(http.DefaultClient, endpoint, authToken)
	}
	return nil
}

func repoFromURL(s string) (string, string, error) {
	parsed, err := url.Parse(s)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse repo from URL %#v: %s", s, err)
	}
	host := parsed.Host
	if strings.HasSuffix(host, "github.com") {
		host = "api." + host
	}
	endpoint := fmt.Sprintf("%s://%s", parsed.Scheme, host)
	return strings.TrimPrefix(strings.TrimSuffix(parsed.Path, ".git"), "/"), endpoint, nil
}
