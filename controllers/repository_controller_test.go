package controllers

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

	pollingv1alpha1 "github.com/bigkevmcd/tekton-polling-operator/api/v1alpha1"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/git"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/pipelines"
	"github.com/bigkevmcd/tekton-polling-operator/pkg/secrets"
	"github.com/google/go-cmp/cmp"
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

var _ reconcile.Reconciler = &RepositoryReconciler{}

func TestRepositoryReconcilerWithEmptyPollState(t *testing.T) {
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

	loaded := &pollingv1alpha1.Repository{}
	err = cl.Get(ctx, req.NamespacedName, loaded)
	fatalIfError(t, err)
	r.PipelineRunner.(*pipelines.MockRunner).AssertPipelineRun(testPipelineName, testRepoURL, testCommitSHA)

	wantStatus := pollingv1alpha1.RepositoryStatus{
		PollStatus: pollingv1alpha1.PollStatus{
			Ref:  "main",
			SHA:  "24317a55785cd98d6c9bf50a5204bc6be17e7316",
			ETag: `W/"878f43039ad0553d0d3122d8bc171b01"`,
		},
	}
	if diff := cmp.Diff(wantStatus, loaded.Status); diff != "" {
		t.Fatalf("incorrect repository status:\n%s", diff)
	}
}

func TestRepositoryReconcilerWithAuthSecret(t *testing.T) {
	authTests := []struct {
		authSecret pollingv1alpha1.AuthSecret
		secretKey  string
	}{
		{
			pollingv1alpha1.AuthSecret{
				SecretReference: corev1.SecretReference{
					Name: testSecretName,
				},
			},
			"token",
		},
		{
			pollingv1alpha1.AuthSecret{
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
		r.PollerFactory = func(_ *pollingv1alpha1.Repository, endpoint, token string) git.CommitPoller {
			if token != testAuthToken {
				t.Fatal("required auth token not provided")
			}
			p := git.NewMockPoller()
			p.AddMockResponse(
				testRepo, pollingv1alpha1.PollStatus{Ref: testRef},
				pollingv1alpha1.PollStatus{Ref: testRef, SHA: testCommitSHA,
					ETag: testCommitETag})
			return p
		}
		req := makeReconcileRequest()
		_, err := r.Reconcile(req)
		fatalIfError(t, err)
	}
}

func TestRepositoryReconcilerErrorPolling(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	repo := makeRepository()
	cl, r := makeReconciler(t, repo, repo)
	req := makeReconcileRequest()
	ctx := context.Background()
	failingErr := errors.New("failing")
	r.PollerFactory = func(*pollingv1alpha1.Repository, string, string) git.CommitPoller {
		p := git.NewMockPoller()
		p.FailWithError(failingErr)
		return p
	}
	_, err := r.Reconcile(req)
	if err != failingErr {
		t.Fatalf("got %#v, want %#v", err, failingErr)
	}

	loaded := &pollingv1alpha1.Repository{}
	err = cl.Get(ctx, req.NamespacedName, loaded)
	fatalIfError(t, err)
	wantStatus := pollingv1alpha1.RepositoryStatus{
		LastError: "failing",
		PollStatus: pollingv1alpha1.PollStatus{
			Ref: "main",
		},
	}
	if diff := cmp.Diff(wantStatus, loaded.Status); diff != "" {
		t.Fatalf("incorrect repository status:\n%s", diff)
	}
	r.PipelineRunner.(*pipelines.MockRunner).RefutePipelineRun(testPipelineName, testRepoURL, testCommitSHA)
}

func TestRepositoryReconcilerWithUnchangedState(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	repo := makeRepository()
	_, r := makeReconciler(t, repo, repo)
	req := makeReconcileRequest()
	_, err := r.Reconcile(req)
	fatalIfError(t, err)
	r.PipelineRunner = pipelines.NewMockRunner(t)

	_, err = r.Reconcile(req)

	fatalIfError(t, err)
	r.PipelineRunner.(*pipelines.MockRunner).RefutePipelineRun(testPipelineName, testRepoURL, testCommitSHA)
}

func TestRepositoryReconcilerClearsLastErrorOnSuccessfulPoll(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	ctx := context.Background()
	repo := makeRepository()
	cl, r := makeReconciler(t, repo, repo)
	failingErr := errors.New("failing")
	savedFactory := r.PollerFactory
	r.PollerFactory = func(*pollingv1alpha1.Repository, string, string) git.CommitPoller {
		p := git.NewMockPoller()
		p.FailWithError(failingErr)
		return p
	}

	req := makeReconcileRequest()
	_, err := r.Reconcile(req)
	if err != failingErr {
		t.Fatalf("got %#v, want %#v", err, failingErr)
	}

	loaded := &pollingv1alpha1.Repository{}
	err = cl.Get(ctx, req.NamespacedName, loaded)
	fatalIfError(t, err)
	if loaded.Status.LastError != "failing" {
		t.Fatalf("got %#v, want %#v", loaded.Status.LastError, "failing")
	}

	r.PollerFactory = savedFactory
	_, err = r.Reconcile(req)
	fatalIfError(t, err)
	fatalIfError(t, cl.Get(ctx, req.NamespacedName, loaded))
}

func makeRepository() *pollingv1alpha1.Repository {
	return &pollingv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRepositoryName,
			Namespace: testRepositoryNamespace,
		},
		Spec: pollingv1alpha1.RepositorySpec{
			URL:       testRepoURL,
			Ref:       testRef,
			Type:      pollingv1alpha1.GitHub,
			Frequency: &metav1.Duration{Duration: testFrequency},
			Pipeline:  pollingv1alpha1.PipelineRef{Name: testPipelineName},
		},
		Status: pollingv1alpha1.RepositoryStatus{},
	}
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

func makeReconciler(t *testing.T, pr *pollingv1alpha1.Repository, objs ...runtime.Object) (client.Client, *RepositoryReconciler) {
	s := scheme.Scheme
	pollingv1alpha1.AddToScheme(s)
	cl := fake.NewFakeClientWithScheme(s, objs...)
	p := git.NewMockPoller()
	p.AddMockResponse(testRepo, pollingv1alpha1.PollStatus{Ref: testRef},
		pollingv1alpha1.PollStatus{Ref: testRef, SHA: testCommitSHA,
			ETag: testCommitETag})
	pollerFactory := func(*pollingv1alpha1.Repository, string, string) git.CommitPoller {
		return p
	}
	return cl, &RepositoryReconciler{
		Client:         cl,
		Scheme:         s,
		PollerFactory:  pollerFactory,
		PipelineRunner: pipelines.NewMockRunner(t),
		SecretGetter:   secrets.New(cl),
		Log:            logf.Log.WithName("testing"),
	}
}

func fatalIfError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
