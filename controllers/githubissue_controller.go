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
	_ = r.Log.WithValues("githubissue", req.NamespacedName)

	// your logic here
	println()
	r.Log.Info("\nENTERED RECONCILE WITH REQ")

	//get the object from the API server
	ghIssue := examplev1alpha1.GitHubIssue{}
	err := r.Client.Get(ctx, req.NamespacedName, &ghIssue)

	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Print("object was deleted (\"not found error\") - return with nil error\n")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	token := os.Getenv("GITHUB_TOKEN")
	issue, _ := findIssue(ghIssue.Spec.Repo, ghIssue.Spec.Title, token)
	fmt.Println("find issue is ok")

	ownerRepo := ghIssue.Spec.Repo
	title := ghIssue.Spec.Title
	description := ghIssue.Spec.Description

	finalizerName := "example.training.redhat.com/finalizer"

	// examine DeletionTimestamp to determine if object is under deletion
	if ghIssue.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(ghIssue.GetFinalizers(), finalizerName) {
			controllerutil.AddFinalizer(&ghIssue, finalizerName)
			if err := r.Update(ctx, &ghIssue); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(ghIssue.GetFinalizers(), finalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(ownerRepo, title, description, string(issue.IssueNumber), token); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&ghIssue, finalizerName)
			if err := r.Update(ctx, &ghIssue); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	//if issue wasn't found (according to title) on github, create it
	if issue == nil {
		//create a git request with github API
		issue, err = create(ownerRepo, title, description, token)
		if err != nil {
			fmt.Printf("create issue failed. error:, %s \n", err)
		} else {
			fmt.Printf("create issue is ok, created issue %s \n", string(issue.IssueNumber))
		}
	}

	//update status fields
	patch := client.MergeFrom(ghIssue.DeepCopy())
	//fmt.Printf("state of object: %s state from web: %s \n", ghIssue.Status.State, issue.State)
	ghIssue.Status.State = issue.State

	ghIssue.Status.LastUpdateTimestamp = issue.LastUpdateTimestamp
	err = r.Client.Status().Patch(ctx, &ghIssue, patch)

	if err != nil {
		fmt.Print("status patch failed \n")
		return ctrl.Result{}, err
	}

	fmt.Printf("title: %s \ndescription: %s\nstatus is: %s \n", ghIssue.Spec.Title, ghIssue.Spec.Description, ghIssue.Status.State)
	fmt.Printf("last updated at %s \n", ghIssue.Status.LastUpdateTimestamp)

	//if title already exists in repo, edit description
	if description != issue.Description {
		//edit description only if there's a difference OR issue was closed
		edit(ownerRepo, title, description, string(issue.IssueNumber), token)
		fmt.Printf("edit issue is ok, edited issue %s \n\n", string(issue.IssueNumber))
	}

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

//function I copied from:
//https://vorozhko.net/create-github-issue-ticket-with-golang

func create(ownerRepo, title, description, token string) (*Issue, error) {
	apiURL := "https://api.github.com/repos/" + ownerRepo + "/issues"
	//title is the only required field
	issueData := NewIssue{Title: title, Description: description}
	//make it json
	jsonData, _ := json.Marshal(issueData)
	//creating client to set custom headers for Authorization
	client := &http.Client{}
	req, _ := http.NewRequest("POST", apiURL, bytes.NewReader(jsonData))

	req.Header.Set("Authorization", "token "+token)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("Response code is is %d\n", resp.StatusCode)
		body, _ := ioutil.ReadAll(resp.Body)
		//print body as it may contain hints in case of errors
		fmt.Println(string(body))
		log.Fatal(err)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	var issue *Issue
	err = json.Unmarshal(body, &issue)
	fmt.Println(string(body))
	return issue, nil
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

func findIssue(ownerRepo, title, token string) (*Issue, error) {
	apiURL := "https://api.github.com/repos/" + ownerRepo + "/issues?state=all"
	//split ownerRepo to owner and repository
	ownerAndRepo := strings.Split(ownerRepo, "/")
	//title is the only required field
	repoData := Repo{Repo: ownerAndRepo[0], Owner: ownerAndRepo[1]}
	//make it json
	jsonData, _ := json.Marshal(repoData)
	//creating client to set custom headers for Authorization
	client := &http.Client{}
	req, _ := http.NewRequest("GET", apiURL, bytes.NewReader(jsonData))
	req.Header.Set("Authorization", "token "+token)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	//print body as it may contain hints in case of errors
	//fmt.Println(string(body))

	var issues []Issue
	err = json.Unmarshal(body, &issues)
	/*loop over issues titles and look for the title given to the function*/
	for _, issue := range issues {
		if issue.Title == title {
			return &issue, nil
		}
	}
	return nil, nil
}

// edit : update issue description
func edit(ownerRepo, title, description, issueNumber, token string) {
	fmt.Printf("editing %s \n", issueNumber)
	apiURL := "https://api.github.com/repos/" + ownerRepo + "/issues/" + issueNumber
	//title is the only required field
	issueData := Issue{Repo: ownerRepo, Title: title, Description: description, IssueNumber: json.Number(issueNumber)}
	//make it json
	jsonData, _ := json.Marshal(issueData)
	//creating client to set custom headers for Authorization
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
		//print body as it may contain hints in case of errors
		fmt.Println(string(body))
		log.Fatal(err)
	}
}

func (r *GitHubIssueReconciler) deleteExternalResources(ownerRepo, title, description, issueNumber, token string) error {
	fmt.Printf("editing %s \n", issueNumber)
	apiURL := "https://api.github.com/repos/" + ownerRepo + "/issues/" + issueNumber
	//title is the only required field
	issueData := Issue{Repo: ownerRepo, Title: title, Description: description, IssueNumber: json.Number(issueNumber), State: "closed"}
	//make it json
	jsonData, _ := json.Marshal(issueData)
	//creating client to set custom headers for Authorization
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
		//print body as it may contain hints in case of errors
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

//func removeString(slice []string, s string) (result []string) {
//	for _, item := range slice {
//		if item == s {
//			continue
//		}
//		result = append(result, item)
//	}
//	return
//}
