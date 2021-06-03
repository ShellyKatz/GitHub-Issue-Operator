package controllers

import (
	"context"
	examplev1alpha1 "github.com/ShellyKatz/example-operator/api/v1alpha1"
	"github.com/ShellyKatz/example-operator/controllers/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

//func newGithubIssueRuntimeObject() runtime.Object{
//	ghIssue := examplev1alpha1.GitHubIssue{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "ghTest",
//			Namespace: "default",
//		},
//		Spec:       examplev1alpha1.GitHubIssueSpec{
//			Repo:        "testUser/testRepo",
//			Title:       "testUnsuccessfulCreate",
//			Description: "testing...",
//		},
//		Status:     examplev1alpha1.GitHubIssueStatus{
//			State:               "",
//			LastUpdateTimestamp: "",
//		},
//	}
//}

func createReconciler(fakeGithubClient *github.FakeClient, fakeK8sClient  client.Client, s *runtime.Scheme ) GitHubIssueReconciler {
	return GitHubIssueReconciler{
		Client:       fakeK8sClient,
		Log:          ctrl.Log.WithName("controllers").WithName("GitHubIssue"),
		Scheme:       s,
		GithubClient: fakeGithubClient,
	}
}

func newFakeK8sClient()  client.Client{
	ghIssue := examplev1alpha1.GitHubIssue{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ghTest",
			Namespace: "default",
					},
		Spec:       examplev1alpha1.GitHubIssueSpec{
			Repo:        "testUser/testRepo",
			Title:       "testUnsuccessfulCreate",
			Description: "testing...",
		},
		Status:     examplev1alpha1.GitHubIssueStatus{
			State:               "",
			LastUpdateTimestamp: "",
		},
	}
	objects := []runtime.Object{ghIssue.DeepCopyObject()}
	fakeK8sClient := fake.NewClientBuilder().WithRuntimeObjects(objects...).Build()
	return fakeK8sClient
}

func TestSuccessfulCreate(t *testing.T){
	t.Skip()
	//reconciler
	//given a valid ghIssue
	//when creating a real issue
	//then reconciler completes and returns ctrl.Result{} and no error
}


func TestUnsuccessfulCreate(t *testing.T){
	//reconciler
	//given we fail to create a real issue
	s := scheme.Scheme
	examplev1alpha1.AddToScheme(s)

	fakeGithubClient := github.NewFakeClient([]github.Issue{})
    fakeGithubClient.FailsOnCreate()

	fakeK8sClient := newFakeK8sClient() //TODO - change function to get the desirable objects list

	r := createReconciler(fakeGithubClient, fakeK8sClient, s)

	//when reconciling
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: "ghTest",
			Namespace: "default",
		},
	}
	_, err := r.Reconcile(context.Background(), req) //TODO what should I do with the "result" val?

	//then reconciler returns ctrl.Result{} and an error
	if err == nil {
		t.Errorf("Expected an error but got nil")
	}

	t.Logf("err is %v", err)
}

//test case: if the issue title already exists in repo
//and both (object and real issue) have the same description - don't create it and don't edit description
func TestCreateAnExistingIssueNoEdit(t *testing.T){
	t.Skip()
}

func TestSuccessfulEdit(t *testing.T){
	t.Skip()
}

func TestUnsuccessfulEdit(t *testing.T){
	t.Skip()
}

func TestSuccessfulDelete(t *testing.T){
	t.Skip()
}

func TestUnsuccessfulDelete(t *testing.T){
	t.Skip()
}

func TestSuccessfulTimeUpdate(t *testing.T){
	t.Skip()
}


func TestUnsuccessfulTimeUpdate(t *testing.T){
	t.Skip()
}

func TestSuccessfulStatusUpdate(t *testing.T){
	t.Skip()
}

func TestUnsuccessfulStatusUpdate(t *testing.T){
	t.Skip()
}




//func TestMemcachedControllerDeploymentCreate(t *testing.T) {
//	var (
//		name            = "githubIssue-operator"
//		namespace       = "githubIssue"
//	)
//	// A Memcached object with metadata and spec.
//	ghIssue := &examplev1alpha1.GitHubIssue{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      name,
//			Namespace: namespace,
//		},
//		Spec: examplev1alpha1.GitHubIssueSpec{
//			Repo: "testUser/TestRepo", // Set desired number of Memcached replicas.
//			Title: "testTrial",
//		},
//	}
//
//	// Objects to track in the fake client.
//	objs := []runtime.Object{ ghIssue.DeepCopy() }
//
//	// Register operator types with the runtime scheme.
//	s := scheme.Scheme
//	examplev1alpha1.AddToScheme(s)
//	//s.AddKnownTypes(examplev1alpha1..SchemeGroupVersion, memcached)
//	//documentation about it
//
//	// Create a fake client to mock API calls.
//	clientBuilder := fake.NewClientBuilder().WithRuntimeObjects(objs...)
//	cl := clientBuilder.Build()
//
//	// Create a ReconcileMemcached object with the scheme and fake client.
//	r := &GitHubIssueReconciler{
//		Client: cl,
//		Scheme: s,//??
//		Log:   ctrl.Log.WithName("controllers").WithName("GitHubIssue"), //??
//		GithubClient: &github.FakeClient{},
//	}
//
//	// Mock request to simulate Reconcile() being called on an event for a
//	// watched resource .
//	req := reconcile.Request{
//		NamespacedName: types.NamespacedName{
//			Name:      name,
//			Namespace: namespace,
//		},
//	}
//
//
//	res, err := r.Reconcile(context.Background(), req)
//	if err != nil {
//		t.Fatalf("reconcile: (%v)", err)
//	}
//	// Check the result of reconciliation to make sure it has the desired state.
//	if !res.Requeue {
//		t.Error("reconcile did not requeue request as expected")
//	}
//	// Check if deployment has been created and has the correct size.
//	dep := &appsv1.Deployment{}
//	err = r.Client.Get(context.TODO(), req.NamespacedName, dep)
//	if err != nil {
//		t.Fatalf("get deployment: (%v)", err)
//	}
//
//}