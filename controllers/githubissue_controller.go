/*
Copyright 2021.
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
	errors2 "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	//"strconv"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	examplev1alpha1 "github.com/ShellyKatz/example-operator/api/v1alpha1"
	//imports for the create function
	"fmt"
	github "github.com/ShellyKatz/example-operator/controllers/github"
)

const FinalizerName = "example.training.redhat.com/finalizer"
const TitleNotFound = "object title not found on github"

// GitHubIssueReconciler reconciles a GitHubIssue object
type GitHubIssueReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	GithubClient github.Client
}

//+kubebuilder:rbac:groups=example.training.redhat.com,resources=githubissues,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=example.training.redhat.com,resources=githubissues/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=example.training.redhat.com,resources=githubissues/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the GitHubIssue object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *GitHubIssueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("githubissue", req.NamespacedName)
	//println("\n#########################################################################\n")
	log.Info("\nENTERED RECONCILE WITH")
	//println("here1")

	//get the object from the API server
	ghIssue := examplev1alpha1.GitHubIssue{}
	err := r.Client.Get(ctx, req.NamespacedName, &ghIssue)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("\nobject was deleted (\"not found error\") - return with nil error")
			//println("ding ding ding")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	//println("here2")

	//bring the issue from the real world (if doesn't exists return nil and err)
	token := os.Getenv("GITHUB_TOKEN")
	issue, findIssueErr := r.GithubClient.FindIssue(ghIssue.Spec, token)
	if findIssueErr != nil && fmt.Sprintf("%v", findIssueErr) != TitleNotFound {
		return ctrl.Result{}, errors2.Wrap(findIssueErr, "error during findIssue")
	}
	log.Info("find issue is ok")
	//println("here3")
	// examine DeletionTimestamp to determine if object is under deletion
	if ghIssue.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// add the finalizer and update the object.
		if !containsString(ghIssue.GetFinalizers(), FinalizerName) {
			if err = r.registerFinalizer(ghIssue, ctx); err != nil {
				return ctrl.Result{}, errors2.Wrap(err, "error during registerFinalizer")
			}
		}
	} else {
		// The object is being deleted
		err := r.deleteGithubIssueObject(ghIssue, issue, findIssueErr, ctx, token)
		return ctrl.Result{}, errors2.Wrap(err, "error during deleteGithubIssueObject")
	}
	//println("here4")
	// if issue wasn't found (according to title) on github, create it
	if fmt.Sprintf("%v", findIssueErr) == TitleNotFound {
		if issue, err = r.GithubClient.Create(ghIssue.Spec, token); err != nil {
			return ctrl.Result{}, errors2.Wrap(err, "error during create")
		} else {
			log.Info("created successfully", "issue number", string(issue.IssueNumber))
		}
	}

	// edit description if needed
	if ghIssue.Spec.Description != issue.Description {
		//edit description only if there's a difference OR issue was closed
		if err = r.GithubClient.Edit(ghIssue.Spec, string(issue.IssueNumber), token); err != nil {
			log.Info("problem here!!!")
			return ctrl.Result{}, errors2.Wrap(err, "error during edit")
		}
		log.Info("edited successfully", "issue number", string(issue.IssueNumber))
	}

	// update status fields
	if err = r.updateStatus(ghIssue, issue, ctx); err != nil {
		return ctrl.Result{}, errors2.Wrap(err, "error during updateStatus")
	}

	fmt.Printf("title: %s \ndescription: %s\nstatus is: %s \n", ghIssue.Spec.Title, ghIssue.Spec.Description, ghIssue.Status.State)
	fmt.Printf("last updated at: %s \n", ghIssue.Status.LastUpdateTimestamp)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GitHubIssueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplev1alpha1.GitHubIssue{}).
		Complete(r)
}

func (r *GitHubIssueReconciler) registerFinalizer(ghIssue examplev1alpha1.GitHubIssue, ctx context.Context) error {
	controllerutil.AddFinalizer(&ghIssue, FinalizerName)
	err := r.Update(ctx, &ghIssue)
	return err
}

//deleteGithubIssueObject: delete the object, it finalizer exists - handle it and then delete object
func (r *GitHubIssueReconciler) deleteGithubIssueObject(ghIssue examplev1alpha1.GitHubIssue, realWorldIssue *github.Issue,
	findIssueErr error, ctx context.Context, token string) error {
	if containsString(ghIssue.GetFinalizers(), FinalizerName) {
		// our finalizer is present, so lets handle any external dependency
		// if the issue isn't on github, skip the external handle and just remove finalizer
		if fmt.Sprintf("%v", findIssueErr) != TitleNotFound {
			if err := r.GithubClient.Close(ghIssue.Spec, string(realWorldIssue.IssueNumber), token); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return err
			}
		}
		// remove our finalizer from the list and update it.
		controllerutil.RemoveFinalizer(&ghIssue, FinalizerName)
		if err := r.Update(ctx, &ghIssue); err != nil {
			return err
		}
	}
	// Stop reconciliation as the item is being deleted
	return nil

}

//deleteExternalResources: close github issue

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func (r *GitHubIssueReconciler) updateStatus(ghIssue examplev1alpha1.GitHubIssue, realWorldIssue *github.Issue,
	ctx context.Context) error {
	patch := client.MergeFrom(ghIssue.DeepCopy())
	ghIssue.Status.State = realWorldIssue.State
	ghIssue.Status.LastUpdateTimestamp = realWorldIssue.LastUpdateTimestamp
	err := r.Client.Status().Patch(ctx, &ghIssue, patch)
	return err
}
