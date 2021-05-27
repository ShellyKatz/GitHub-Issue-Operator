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
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	examplev1alpha1 "github.com/ShellyKatz/example-operator/api/v1alpha1"
	//imports for the create function
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

const FinalizerName = "example.training.redhat.com/finalizer"
const TitleNotFound = "object title not found on github"

// GitHubIssueReconciler reconciles a GitHubIssue object
type GitHubIssueReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
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
	println("\n#########################################################################\n")
	log.Info("\nENTERED RECONCILE WITH")

	//get the object from the API server
	ghIssue := examplev1alpha1.GitHubIssue{}
	err := r.Client.Get(ctx, req.NamespacedName, &ghIssue)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("\nobject was deleted (\"not found error\") - return with nil error")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//bring the issue from the real world (if doesn't exists return nil and err)
	token := os.Getenv("GITHUB_TOKEN")

	issue, findIssueErr := findIssue(ghIssue.Spec, token)
	if findIssueErr != nil && fmt.Sprintf("%v", findIssueErr) != TitleNotFound {
		return ctrl.Result{}, errors2.Wrap(findIssueErr, "error during findIssue")
	}
	log.Info("find issue is ok")

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

	// if issue wasn't found (according to title) on github, create it
	if fmt.Sprintf("%v", findIssueErr) == TitleNotFound {
		if issue, err = create(ghIssue.Spec, token); err != nil {
			return ctrl.Result{}, errors2.Wrap(err, "error during create")
		} else {
			log.Info("created successfully", "issue number", string(issue.IssueNumber))
		}
	}

	// edit description if needed
	if ghIssue.Spec.Description != issue.Description {
		//edit description only if there's a difference OR issue was closed
		if err = edit(ghIssue.Spec, string(issue.IssueNumber), token); err != nil {
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

// NewIssue https://vorozhko.net/create-github-issue-ticket-with-golang
// specify data fields for new github issue submission
type NewIssue struct {
	Title       string `json:"title"`
	Description string `json:"body"`
}

// function I copied from:
// https://vorozhko.net/create-github-issue-ticket-with-golang

func create(ghIssueSpec examplev1alpha1.GitHubIssueSpec, token string) (*Issue, error) {
	apiURL := "https://api.github.com/repos/" + ghIssueSpec.Repo + "/issues"
	// title is the only required field
	issueData := NewIssue{Title: ghIssueSpec.Title, Description: ghIssueSpec.Description}
	// make it json
	jsonData, _ := json.Marshal(issueData)
	// creating client to set custom headers for Authorization
	client := &http.Client{}
	req, _ := http.NewRequest("POST", apiURL, bytes.NewReader(jsonData))

	req.Header.Set("Authorization", "token "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("Response code is is %d\n", resp.StatusCode)
		body, _ := ioutil.ReadAll(resp.Body)
		//print body as it may contain hints in case of errors
		fmt.Println(string(body))
		return nil, err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	var issue *Issue
	err = json.Unmarshal(body, &issue)

	//fmt.Println(string(body))
	return issue, err
}

type Repo struct {
	Repo  string `json:"repo"`
	Owner string `json:"owner"`
}

type Issue struct {
	Repo                string      `json:"repo"`
	Title               string      `json:"title"`
	Description         string      `json:"body"`
	IssueNumber         json.Number `json:"number,omitempty"`
	State               string      `json:"state,omitempty"`
	LastUpdateTimestamp string      `json:"updated_at"`
}

func findIssue(ghIssueSpec examplev1alpha1.GitHubIssueSpec, token string) (*Issue, error) {
	apiURL := "https://api.github.com/repos/" + ghIssueSpec.Repo + "/issues?state=all"
	// split ownerRepo to owner and repository
	ownerAndRepo := strings.Split(ghIssueSpec.Repo, "/")
	// title is the only required field
	repoData := Repo{Repo: ownerAndRepo[0], Owner: ownerAndRepo[1]}
	// make it json
	jsonData, _ := json.Marshal(repoData)
	// creating client to set custom headers for Authorization
	client := &http.Client{}
	req, _ := http.NewRequest("GET", apiURL, bytes.NewReader(jsonData))
	req.Header.Set("Authorization", "token "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	// print body as it may contain hints in case of errors
	// fmt.Println(string(body))

	var issues []Issue
	err = json.Unmarshal(body, &issues)

	if err != nil {
		return nil, err
	}
	// loop over issues titles and look for the title given to the function
	for _, issue := range issues {
		if issue.Title == ghIssueSpec.Title {
			return &issue, nil
		}
	}
	err = fmt.Errorf(TitleNotFound)
	return nil, err
}

// edit : update issue description
func edit(ghIssueSpec examplev1alpha1.GitHubIssueSpec, issueNumber, token string) error {
	fmt.Printf("editing %s \n", issueNumber)
	apiURL := "https://api.github.com/repos/" + ghIssueSpec.Repo + "/issues/" + issueNumber
	// title is the only required field
	issueData := Issue{Repo: ghIssueSpec.Repo, Title: ghIssueSpec.Title, Description: ghIssueSpec.Description,
		IssueNumber: json.Number(issueNumber)}
	// make it json
	jsonData, _ := json.Marshal(issueData)
	// creating client to set custom headers for Authorization
	client := &http.Client{}
	req, _ := http.NewRequest("PATCH", apiURL, bytes.NewReader(jsonData))
	req.Header.Set("Authorization", "token "+token)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Response code is %d\n", resp.StatusCode)
		body, _ := ioutil.ReadAll(resp.Body)
		// print body as it may contain hints in case of errors
		fmt.Println(string(body))
		return err
	}

	return nil
}

func (r *GitHubIssueReconciler) registerFinalizer(ghIssue examplev1alpha1.GitHubIssue, ctx context.Context) error {
	controllerutil.AddFinalizer(&ghIssue, FinalizerName)
	err := r.Update(ctx, &ghIssue)
	return err
}

//deleteGithubIssueObject: delete the object, it finalizer exists - handle it and then delete object
func (r *GitHubIssueReconciler) deleteGithubIssueObject(ghIssue examplev1alpha1.GitHubIssue, realWorldIssue *Issue,
	findIssueErr error, ctx context.Context, token string) error {
	if containsString(ghIssue.GetFinalizers(), FinalizerName) {
		// our finalizer is present, so lets handle any external dependency
		// if the issue isn't on github, skip the external handle and just remove finalizer
		if fmt.Sprintf("%v", findIssueErr) != TitleNotFound {
			if err := r.deleteExternalResources(ghIssue.Spec, string(realWorldIssue.IssueNumber), token); err != nil {
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
func (r *GitHubIssueReconciler) deleteExternalResources(ghIssueSpec examplev1alpha1.GitHubIssueSpec,
	issueNumber, token string) error {
	apiURL := "https://api.github.com/repos/" + ghIssueSpec.Repo + "/issues/" + issueNumber
	// title is the only required field
	issueData := Issue{Repo: ghIssueSpec.Repo, Title: ghIssueSpec.Title, Description: ghIssueSpec.Description,
		IssueNumber: json.Number(issueNumber), State: "closed"}
	// make it json
	jsonData, _ := json.Marshal(issueData)
	// creating client to set custom headers for Authorization
	client := &http.Client{}
	req, _ := http.NewRequest("PATCH", apiURL, bytes.NewReader(jsonData))

	req.Header.Set("Authorization", "token "+token)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Response code is %d\n", resp.StatusCode)
		body, _ := ioutil.ReadAll(resp.Body)
		// print body as it may contain hints in case of errors
		fmt.Println(string(body))
		log.Fatal(err)
	}

	return nil
}

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func (r *GitHubIssueReconciler) updateStatus(ghIssue examplev1alpha1.GitHubIssue, realWorldIssue *Issue,
	ctx context.Context) error {
	patch := client.MergeFrom(ghIssue.DeepCopy())
	ghIssue.Status.State = realWorldIssue.State
	ghIssue.Status.LastUpdateTimestamp = realWorldIssue.LastUpdateTimestamp
	err := r.Client.Status().Patch(ctx, &ghIssue, patch)
	return err
}
