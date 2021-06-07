package controllers

import (
	"context"
	examplev1alpha1 "github.com/ShellyKatz/example-operator/api/v1alpha1"
	"github.com/ShellyKatz/example-operator/controllers/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
	"time"
)

var s = createAndAddScheme()

func newGithubIssueRuntimeObject(title, description, state, lastUpdateTimeStamp string, finalizersList []string,
								 isBeingDeleted bool) examplev1alpha1.GitHubIssue{
	ghIssueObj := examplev1alpha1.GitHubIssue{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ghTest",
			Namespace: "default",
			Finalizers: finalizersList,
		},
		Spec: examplev1alpha1.GitHubIssueSpec{
			Repo:        "testUser/testRepo",
			Title:       title,
			Description: description,
		},
		Status: examplev1alpha1.GitHubIssueStatus{
			State:               state,
			LastUpdateTimestamp: lastUpdateTimeStamp,
		},
	}
	//add deletion timestamp
	if isBeingDeleted == true {
		time := v1.Time{
			Time: time.Now(),
		}
		ghIssueObj.SetDeletionTimestamp(&time)
	}
	return ghIssueObj
}

func createReconciler(fakeGithubClient *github.FakeClient, fakeK8sClient client.Client, s *runtime.Scheme) GitHubIssueReconciler {
	return GitHubIssueReconciler{
		Client:       fakeK8sClient,
		Log:          ctrl.Log.WithName("controllers").WithName("GitHubIssue"),
		Scheme:       s,
		GithubClient: fakeGithubClient,
	}
}

func newFakeK8sClient(objects... examplev1alpha1.GitHubIssue) client.Client {
	var runtimeObjects []runtime.Object
	for _, obj := range objects {
		runtimeObjects = append(runtimeObjects, obj.DeepCopy())
	}
	fakeK8sClient := fake.NewClientBuilder().WithRuntimeObjects(runtimeObjects...).Build()
	return fakeK8sClient
}

func createReq() reconcile.Request{
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "ghTest",
			Namespace: "default",
		},
	}
}

func createAndAddScheme() *runtime.Scheme{
	s := scheme.Scheme
	examplev1alpha1.AddToScheme(s)
	return s
}

func createFakeGithubIssue()  github.Issue{
	return github.Issue{
		Repo:                "testUser/testRepo",
		Title:               "testIssue",
		Description:         "testing...",
		IssueNumber:         "1",
		State:               "open",
		LastUpdateTimestamp: "2021-05-31T07:49:28Z",
	}
}

func TestSuccessfulCreate(t *testing.T) {
	//given an empty repository and a valid ghIssue object
	fakeGithubClient := github.NewFakeClient([]*github.Issue{}, false, "no error")

	ghIssueObj := newGithubIssueRuntimeObject("testIssue", "testing...", "", "",
		                                       []string{},false)
	fakeK8sClient := newFakeK8sClient(ghIssueObj)

	r := createReconciler(fakeGithubClient, fakeK8sClient, s)

	//when reconciling
	_, err := r.Reconcile(context.Background(), createReq()) //TODO what should I do with the "result" val?

	//then reconciler completes and returns ctrl.Result{} and no error
	if err != nil {
		t.Errorf("Expected no error but got an error: %v", err)
	}
	if fakeGithubClient.Issues[0].State != "open" {
		t.Errorf("Expected state \"open\" but got: %s", fakeGithubClient.Issues[0].State)
	}
	if fakeGithubClient.Issues[0].LastUpdateTimestamp != "2021-05-31T07:49:28Z" {
		t.Errorf("Expected LastUpdateTimestamp \"2021-05-31T07:49:28Z\" but got: %s", fakeGithubClient.Issues[0].LastUpdateTimestamp)
	}
	t.Logf(fakeGithubClient.Issues[0].LastUpdateTimestamp)
}

func TestUnsuccessfulCreate(t *testing.T) {
	//given a ghIssue object and a fake github client that always fails on create
	fakeGithubClient := github.NewFakeClient([]*github.Issue{}, true, github.CreatError)

	ghIssueObj := newGithubIssueRuntimeObject("testIssue", "testing...", "", "",
		                                       []string{},false)
	fakeK8sClient := newFakeK8sClient(ghIssueObj)

	r := createReconciler(fakeGithubClient, fakeK8sClient, s)

	//when reconciling
	_, err := r.Reconcile(context.Background(), createReq()) //TODO what should I do with the "result" val?

	//then reconciler returns ctrl.Result{} and an error
	if err == nil {
		t.Errorf("Expected an error but got nil")
	} else {
		t.Logf("err is %v", err)
	}
}

func TestCreateAnExistingIssueNoEdit(t *testing.T) {
	//given a valid ghIssue object that it's title already appears in repo and have the same description
	issue := createFakeGithubIssue()

	fakeRepo := []*github.Issue{&issue}
	fakeGithubClient := github.NewFakeClient(fakeRepo, false, "no error")

	ghIssueObj := newGithubIssueRuntimeObject("testIssue", "testing...",
		                                 "", "", []string{}, false)
	fakeK8sClient := newFakeK8sClient(ghIssueObj)

	r := createReconciler(fakeGithubClient, fakeK8sClient, s)

	//when reconciling
	_, err := r.Reconcile(context.Background(), createReq()) //TODO what should I do with the "result" val?

	//then reconciler completes and returns ctrl.Result{} no error and object isn't added to repo
	if err != nil {
		t.Errorf("Expected no error but got an error: %v", err)
	}
	if len(fakeGithubClient.Issues) != 1 {
		t.Errorf("Expected repo to stay in len 1 but got len: %d", len(fakeGithubClient.Issues))
	}
}

func TestSuccessfulEdit(t *testing.T) {
	//given a valid ghIssue that it's title already appears in repo but with different description
	issue := createFakeGithubIssue()

	fakeRepo := []*github.Issue{&issue}
	fakeGithubClient := github.NewFakeClient(fakeRepo, false, "no error")

	ghIssueObj := newGithubIssueRuntimeObject("testIssue", "testing...edit!", "", "",
												[]string{},false)
	fakeK8sClient := newFakeK8sClient(ghIssueObj)

	r := createReconciler(fakeGithubClient, fakeK8sClient, s)

	//when reconciling
	_, err := r.Reconcile(context.Background(), createReq()) //TODO what should I do with the "result" val?

	//then reconciler completes and returns ctrl.Result{} no error, object isn't added to repo
	//and the github issue's description (in "real world") is edited
	if err != nil {
		t.Errorf("Expected no error but got an error: %v", err)
	}
	if len(fakeGithubClient.Issues) != 1 {
		t.Errorf("Expected repo to stay in len 1 but got len: %d", len(fakeGithubClient.Issues))
	}
	if fakeGithubClient.Issues[0].Description != "testing...edit!" {
		t.Errorf("Expected description to be: testing...edit! but got: %s",
			fakeGithubClient.Issues[0].Description)
	}
}

func TestUnsuccessfulEdit(t *testing.T) {
	//given a ghIssue object and a fake github client that always fails on edit
	issue := createFakeGithubIssue()

	fakeRepo := []*github.Issue{&issue}
	fakeGithubClient := github.NewFakeClient(fakeRepo, true, github.EditError)

	ghIssueObj := newGithubIssueRuntimeObject("testIssue", "testing...edit!", "", "",
												[]string{} ,false)
	fakeK8sClient := newFakeK8sClient(ghIssueObj)

	r := createReconciler(fakeGithubClient, fakeK8sClient, s)

	//when reconciling
	_, err := r.Reconcile(context.Background(), createReq()) //TODO what should I do with the "result" val?

	//then reconciler completes and returns ctrl.Result{} and an error
	if err == nil {
		t.Errorf("Expected an error but got nil")
	} else {
		t.Logf("err is %v", err)
	}
}

func TestSuccessfulDelete(t *testing.T) {
	//given a valid ghIssue with a deletion timestamp and a finalizer
	issue := createFakeGithubIssue()

	fakeRepo := []*github.Issue{&issue}
	fakeGithubClient := github.NewFakeClient(fakeRepo, false, "no error")

	ghIssueObj := newGithubIssueRuntimeObject("testIssue", "testing...", "", "",
												[]string{FinalizerName} ,true)
	fakeK8sClient := newFakeK8sClient(ghIssueObj)

	r := createReconciler(fakeGithubClient, fakeK8sClient, s)

	//when reconciling
	_, err := r.Reconcile(context.Background(), createReq()) //TODO what should I do with the "result" val?

	//then reconciler completes and returns ctrl.Result{} no error and object's state becomes "closed"
	if err != nil {
		t.Errorf("Expected no error but got an error: %v", err)
	}
	if fakeGithubClient.Issues[0].State!= "closed" {
		t.Errorf("Expected issue's state to be \"closed\" but got: %s", fakeGithubClient.Issues[0].State)
	}
}

func TestUnsuccessfulDelete(t *testing.T) {
	//given a unValid ghIssue that in repo
	issue := createFakeGithubIssue()
	fakeRepo := []*github.Issue{&issue}
	fakeGithubClient := github.NewFakeClient(fakeRepo, true, github.DeleteError)

	ghIssueObj := newGithubIssueRuntimeObject("testIssue", "testing...", "", "",
												[]string{FinalizerName} ,true)
	fakeK8sClient := newFakeK8sClient(ghIssueObj)

	r := createReconciler(fakeGithubClient, fakeK8sClient, s)

	//when reconciling
	_, err := r.Reconcile(context.Background(), createReq()) //TODO what should I do with the "result" val?

	//then reconciler completes and returns ctrl.Result{} no error and object isn't added to repo
	if err == nil {
		t.Errorf("Expected an error but got nil")
	} else {
		t.Logf("err is %v", err)
	}
}

func TestSuccessfulTimeUpdate(t *testing.T) {
	t.Skip()
}

func TestUnsuccessfulTimeUpdate(t *testing.T) {
	t.Skip()
}

func TestSuccessfulStatusUpdate(t *testing.T) {
	t.Skip()
}

func TestUnsuccessfulStatusUpdate(t *testing.T) {
	t.Skip()
}