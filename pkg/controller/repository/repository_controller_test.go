package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	pollingv1 "github.com/bigkevmcd/tekton-polling-operator/pkg/apis/polling/v1alpha1"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/git"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/pipelines"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/secrets"
	"github.com/google/go-cmp/cmp"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const (
	testFrequency           = time.Second * 10
	testRepositoryName      = "test-repository"
	testRepositoryNamespace = "test-repository-ns"
	testRepoURL             = "https://github.com/example/example.git"
	testRepo                = "example/example"
	testRef                 = "main"
	testSecretName          = "test-secret"
	testAuthToken           = "test-auth-token"
	testCommitSHA           = "24317a55785cd98d6c9bf50a5204bc6be17e7316"
	testCommitETag          = `W/"878f43039ad0553d0d3122d8bc171b01"`
	testPipelineName        = "test-pipeline"
)

var _ reconcile.Reconciler = &ReconcileRepository{}

func TestReconcileRepositoryWithEmptyPollState(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	repo := makeRepository()
	cl, r := makeReconciler(t, repo, repo)
	req := makeReconcileRequest()
	ctx := context.Background()

	res, err := r.Reconcile(req)
	fatalIfError(t, err)
	wantResult := reconcile.Result{
		RequeueAfter: time.Second * 10,
	}
	if diff := cmp.Diff(wantResult, res); diff != "" {
		t.Fatalf("reconciliation result is different:\n%s", diff)
	}

	loaded := &pollingv1.Repository{}
	err = cl.Get(ctx, req.NamespacedName, loaded)
	fatalIfError(t, err)
	r.pipelineRunner.(*pipelines.MockRunner).AssertPipelineRun(
		testPipelineName, testRepositoryNamespace,
		makeTestParams(map[string]string{"one": testRepoURL, "two": "main"}))

	wantStatus := pollingv1.RepositoryStatus{
		PollStatus: pollingv1.PollStatus{
			Ref:  "main",
			SHA:  "24317a55785cd98d6c9bf50a5204bc6be17e7316",
			ETag: `W/"878f43039ad0553d0d3122d8bc171b01"`,
		},
	}
	if diff := cmp.Diff(wantStatus, loaded.Status); diff != "" {
		t.Fatalf("incorrect repository status:\n%s", diff)
	}
}

func TestReconcileRepositoryInPipelineNamespace(t *testing.T) {
	pipelineNS := "test-pipeline-ns"
	logf.SetLogger(logf.ZapLogger(true))
	repo := makeRepository(func(r *pollingv1.Repository) {
		r.Spec.Pipeline.Namespace = pipelineNS
	})
	cl, r := makeReconciler(t, repo, repo)
	req := makeReconcileRequest()
	ctx := context.Background()

	res, err := r.Reconcile(req)
	fatalIfError(t, err)
	wantResult := reconcile.Result{
		RequeueAfter: time.Second * 10,
	}
	if diff := cmp.Diff(wantResult, res); diff != "" {
		t.Fatalf("reconciliation result is different:\n%s", diff)
	}

	loaded := &pollingv1.Repository{}
	err = cl.Get(ctx, req.NamespacedName, loaded)
	fatalIfError(t, err)
	r.pipelineRunner.(*pipelines.MockRunner).AssertPipelineRun(
		testPipelineName, pipelineNS,
		makeTestParams(map[string]string{"one": testRepoURL, "two": "main"}))

	wantStatus := pollingv1.RepositoryStatus{
		PollStatus: pollingv1.PollStatus{
			Ref:  "main",
			SHA:  "24317a55785cd98d6c9bf50a5204bc6be17e7316",
			ETag: `W/"878f43039ad0553d0d3122d8bc171b01"`,
		},
	}
	if diff := cmp.Diff(wantStatus, loaded.Status); diff != "" {
		t.Fatalf("incorrect repository status:\n%s", diff)
	}
}

func TestReconcileRepositoryWithAuthSecret(t *testing.T) {
	authTests := []struct {
		authSecret pollingv1.AuthSecret
		secretKey  string
	}{
		{
			pollingv1.AuthSecret{
				SecretReference: corev1.SecretReference{
					Name: testSecretName,
				},
			},
			"token",
		},
		{
			pollingv1.AuthSecret{
				SecretReference: corev1.SecretReference{
					Name: testSecretName,
				},
				Key: "custom-key",
			},
			"custom-key",
		},
	}

	for _, tt := range authTests {
		logf.SetLogger(logf.ZapLogger(true))
		repo := makeRepository()
		repo.Spec.Auth = &tt.authSecret
		_, r := makeReconciler(t, repo, repo, makeTestSecret(testSecretName, tt.secretKey))
		r.pollerFactory = func(_ *pollingv1.Repository, endpoint, token string) git.CommitPoller {
			if token != testAuthToken {
				t.Fatal("required auth token not provided")
			}
			p := git.NewMockPoller()
			p.AddMockResponse(
				testRepo, pollingv1.PollStatus{Ref: testRef},
				map[string]interface{}{"id": testRef},
				pollingv1.PollStatus{Ref: testRef, SHA: testCommitSHA,
					ETag: testCommitETag})
			return p
		}
		req := makeReconcileRequest()
		_, err := r.Reconcile(req)
		fatalIfError(t, err)
	}
}

func TestReconcileRepositoryErrorPolling(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	repo := makeRepository()
	cl, r := makeReconciler(t, repo, repo)
	req := makeReconcileRequest()
	ctx := context.Background()
	failingErr := errors.New("failing")
	r.pollerFactory = func(*pollingv1.Repository, string, string) git.CommitPoller {
		p := git.NewMockPoller()
		p.FailWithError(failingErr)
		return p
	}
	_, err := r.Reconcile(req)
	if err != failingErr {
		t.Fatalf("got %#v, want %#v", err, failingErr)
	}

	loaded := &pollingv1.Repository{}
	err = cl.Get(ctx, req.NamespacedName, loaded)
	fatalIfError(t, err)
	wantStatus := pollingv1.RepositoryStatus{
		LastError: "failing",
		PollStatus: pollingv1.PollStatus{
			Ref: "main",
		},
	}
	if diff := cmp.Diff(wantStatus, loaded.Status); diff != "" {
		t.Fatalf("incorrect repository status:\n%s", diff)
	}
	r.pipelineRunner.(*pipelines.MockRunner).AssertNoPipelineRuns()
}

func TestReconcileRepositoryWithUnchangedState(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	repo := makeRepository()
	_, r := makeReconciler(t, repo, repo)
	p := git.NewMockPoller()
	p.AddMockResponse(testRepo, pollingv1.PollStatus{Ref: testRef},
		map[string]interface{}{"id": testRef},
		pollingv1.PollStatus{Ref: testRef, SHA: testCommitSHA,
			ETag: testCommitETag})
	p.AddMockResponse(
		testRepo, pollingv1.PollStatus{Ref: testRef, SHA: testCommitSHA,
			ETag: testCommitETag},
		nil,
		pollingv1.PollStatus{Ref: testRef, SHA: testCommitSHA,
			ETag: testCommitETag})

	r.pollerFactory = func(_ *pollingv1.Repository, endpoint, token string) git.CommitPoller {
		return p
	}

	req := makeReconcileRequest()
	_, err := r.Reconcile(req)
	fatalIfError(t, err)
	r.pipelineRunner = pipelines.NewMockRunner(t)

	_, err = r.Reconcile(req)

	fatalIfError(t, err)
	r.pipelineRunner.(*pipelines.MockRunner).AssertNoPipelineRuns()
}

func TestReconcileRepositoryClearsLastErrorOnSuccessfulPoll(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	ctx := context.Background()
	repo := makeRepository()
	cl, r := makeReconciler(t, repo, repo)
	failingErr := errors.New("failing")
	savedFactory := r.pollerFactory
	r.pollerFactory = func(*pollingv1.Repository, string, string) git.CommitPoller {
		p := git.NewMockPoller()
		p.FailWithError(failingErr)
		return p
	}

	req := makeReconcileRequest()
	_, err := r.Reconcile(req)
	if err != failingErr {
		t.Fatalf("got %#v, want %#v", err, failingErr)
	}

	loaded := &pollingv1.Repository{}
	err = cl.Get(ctx, req.NamespacedName, loaded)
	fatalIfError(t, err)
	if loaded.Status.LastError != "failing" {
		t.Fatalf("got %#v, want %#v", loaded.Status.LastError, "failing")
	}

	r.pollerFactory = savedFactory
	_, err = r.Reconcile(req)
	fatalIfError(t, err)
	fatalIfError(t, cl.Get(ctx, req.NamespacedName, loaded))
}

func makeRepository(opts ...func(*pollingv1.Repository)) *pollingv1.Repository {
	r := &pollingv1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRepositoryName,
			Namespace: testRepositoryNamespace,
		},
		Spec: pollingv1.RepositorySpec{
			URL:       testRepoURL,
			Ref:       testRef,
			Type:      pollingv1.GitHub,
			Frequency: &metav1.Duration{Duration: testFrequency},
			Pipeline: pollingv1.PipelineRef{Name: testPipelineName, Params: []pollingv1.Param{
				{Name: "one", Expression: "repoURL"},
				{Name: "two", Expression: "commit.id"},
			}},
		},
		Status: pollingv1.RepositoryStatus{},
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

func makeReconcileRequest() reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      testRepositoryName,
			Namespace: testRepositoryNamespace,
		},
	}
}

func makeTestSecret(n, key string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      n,
			Namespace: testRepositoryNamespace,
		},
		Data: map[string][]byte{
			key: []byte(testAuthToken),
		},
	}
}

func makeReconciler(t *testing.T, pr *pollingv1.Repository, objs ...runtime.Object) (client.Client, *ReconcileRepository) {
	s := scheme.Scheme
	s.AddKnownTypes(pollingv1.SchemeGroupVersion, pr)
	cl := fake.NewFakeClientWithScheme(s, objs...)
	p := git.NewMockPoller()
	p.AddMockResponse(testRepo, pollingv1.PollStatus{Ref: testRef},
		map[string]interface{}{"id": testRef},
		pollingv1.PollStatus{Ref: testRef, SHA: testCommitSHA,
			ETag: testCommitETag})
	pollerFactory := func(*pollingv1.Repository, string, string) git.CommitPoller {
		return p
	}
	return cl, &ReconcileRepository{
		client:         cl,
		scheme:         s,
		pollerFactory:  pollerFactory,
		pipelineRunner: pipelines.NewMockRunner(t),
		secretGetter:   secrets.New(cl),
		log:            logf.Log.WithName("testing"),
	}
}

func fatalIfError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func makeTestParams(vars map[string]string) []pipelinev1beta1.Param {
	params := []pipelinev1beta1.Param{}
	for k, v := range vars {
		params = append(params, pipelinev1beta1.Param{
			Name: k, Value: pipelinev1beta1.NewArrayOrString(v)})
	}
	return params
}
