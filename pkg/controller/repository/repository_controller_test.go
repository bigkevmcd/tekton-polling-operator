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

var _ reconcile.Reconciler = &ReconcileRepository{}

func TestReconcileRepositoryWithEmptyPollState(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	repo := makeRepository()
	cl, r := makeReconciler(t, repo, repo, makeTestSecret(testSecretName))
	req := makeReconcileRequest()
	ctx := context.Background()

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatal(err)
	}
	wantResult := reconcile.Result{
		RequeueAfter: time.Second * 10,
	}
	if diff := cmp.Diff(wantResult, res); diff != "" {
		t.Fatalf("reconciliation result is different:\n%s", diff)
	}

	loaded := &pollingv1.Repository{}
	err = cl.Get(ctx, req.NamespacedName, loaded)
	if err != nil {
		t.Fatal(err)
	}

	r.pipelineRunner.(*pipelines.MockRunner).AssertPipelineRun(testPipelineName, testRepoURL, testCommitSHA)

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
	t.Skip()
}

func TestReconcileRepositoryErrorPolling(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	repo := makeRepository()
	cl, r := makeReconciler(t, repo, repo, makeTestSecret(testSecretName))
	req := makeReconcileRequest()
	ctx := context.Background()
	failingErr := errors.New("failing")
	r.poller.(*git.MockPoller).FailWithError(failingErr)

	res, err := r.Reconcile(req)
	if err != failingErr {
		t.Fatal(err)
	}
	wantResult := reconcile.Result{
		Requeue: true,
	}
	if diff := cmp.Diff(wantResult, res); diff != "" {
		t.Fatalf("reconciliation result is different:\n%s", diff)
	}

	loaded := &pollingv1.Repository{}
	err = cl.Get(ctx, req.NamespacedName, loaded)
	if err != nil {
		t.Fatal(err)
	}
	wantStatus := pollingv1.RepositoryStatus{
		LastError: "failing",
		PollStatus: pollingv1.PollStatus{
			Ref: "main",
		},
	}
	if diff := cmp.Diff(wantStatus, loaded.Status); diff != "" {
		t.Fatalf("incorrect repository status:\n%s", diff)
	}
	r.pipelineRunner.(*pipelines.MockRunner).RefutePipelineRun(testPipelineName, testRepoURL, testCommitSHA)
}

func TestReconcileRepositoryWithUnchangedState(t *testing.T) {
	t.Skip()
}

func makeRepository() *pollingv1.Repository {
	return &pollingv1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRepositoryName,
			Namespace: testRepositoryNamespace,
		},
		Spec: pollingv1.RepositorySpec{
			URL: testRepoURL,
			Ref: testRef,
			Auth: pollingv1.AuthSecret{
				SecretReference: corev1.SecretReference{
					Name: testSecretName,
				},
			},
			Type:      pollingv1.GitHub,
			Frequency: &metav1.Duration{Duration: testFrequency},
			Pipeline:  pollingv1.PipelineRef{Name: testPipelineName},
		},
		Status: pollingv1.RepositoryStatus{},
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

func makeTestSecret(n string) *corev1.Secret {
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
			"token": []byte(testAuthToken),
		},
	}
}

func makeReconciler(t *testing.T, pr *pollingv1.Repository, objs ...runtime.Object) (client.Client, *ReconcileRepository) {
	s := scheme.Scheme
	s.AddKnownTypes(pollingv1.SchemeGroupVersion, pr)
	cl := fake.NewFakeClientWithScheme(s, objs...)
	// TODO: reorganise this to make it easier to pass in.
	p := git.NewMockPoller()
	p.AddMockResponse(testRepo, pollingv1.PollStatus{Ref: testRef}, &pollingv1.PollStatus{Ref: testRef, SHA: testCommitSHA, ETag: testCommitETag})
	return cl, &ReconcileRepository{
		client:         cl,
		scheme:         s,
		poller:         p,
		pipelineRunner: pipelines.NewMockRunner(t),
	}
}
