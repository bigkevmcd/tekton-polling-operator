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

	pollingv1 "github.com/bigkevmcd/tekton-polling-operator/pkg/apis/polling/v1alpha1"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/cel"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/git"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/pipelines"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/secrets"
	"github.com/go-logr/logr"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
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
		pipelineRunner: pipelines.NewRunner(mgr.GetClient()),
		secretGetter:   secrets.New(mgr.GetClient()),
		log:            logf.Log.WithName("controller_repository"),
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
	// The pipelineRunner executes the named pipeline with appropriate params.
	pipelineRunner pipelines.PipelineRunner
	secretGetter   secrets.SecretGetter
	log            logr.Logger
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
	runNS := repo.Spec.Pipeline.Namespace
	if runNS == "" {
		runNS = req.Namespace
	}

	params, err := makeParams(commit, repo.Spec)
	if err != nil {
		reqLogger.Error(err, "failed to parse the parameters")
		return reconcile.Result{}, err
	}
	pr, err := r.pipelineRunner.Run(ctx, repo.Spec.Pipeline.Name, runNS, params, repo.Spec.Pipeline.Resources)
	if err != nil {
		reqLogger.Error(err, "failed to create a PipelineRun", "pipelineName", repo.Spec.Pipeline.Name)
		return reconcile.Result{}, err
	}
	reqLogger.Info("PipelineRun created", "name", pr.ObjectMeta.Name)
	reqLogger.Info("Requeueing next check", "frequency", repo.GetFrequency())
	return reconcile.Result{RequeueAfter: repo.GetFrequency()}, nil
}

func (r *ReconcileRepository) authTokenForRepo(ctx context.Context, logger logr.Logger, namespace string, repo *pollingv1.Repository) (string, error) {
	if repo.Spec.Auth == nil {
		return "", nil
	}
	key := "token"
	if repo.Spec.Auth.Key != "" {
		key = repo.Spec.Auth.Key
	}
	authToken, err := r.secretGetter.SecretToken(ctx, types.NamespacedName{Name: repo.Spec.Auth.Name, Namespace: namespace}, key)
	if err != nil {
		logger.Error(err, "Getting the auth token failed", "name", repo.Spec.Auth.Name, "namespace", namespace, "key", key)
		return "", err
	}
	return authToken, nil
}

func makeParams(commit git.Commit, spec pollingv1.RepositorySpec) ([]pipelinev1.Param, error) {
	celctx, err := cel.New(spec.URL, commit)
	if err != nil {
		return nil, err
	}
	params := []pipelinev1.Param{}
	for _, v := range spec.Pipeline.Params {
		val, err := celctx.EvaluateToParamValue(v.Expression)
		if err != nil {
			return nil, err
		}
		params = append(params, pipelinev1.Param{Name: v.Name, Value: *val})
	}
	return params, nil
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
