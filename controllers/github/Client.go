package github

import (
	"encoding/json"
	examplev1alpha1 "github.com/ShellyKatz/example-operator/api/v1alpha1"
)

const TitleNotFound = "object title not found on github" //error for findIssue function

type Client interface {
	FindIssue(ghIssueSpec examplev1alpha1.GitHubIssueSpec, token string) (*Issue, error)
	Create(ghIssueSpec examplev1alpha1.GitHubIssueSpec, token string) (*Issue, error)
	Edit(ghIssueSpec examplev1alpha1.GitHubIssueSpec, issueNumber, token string) error
	Close(ghIssueSpec examplev1alpha1.GitHubIssueSpec, issueNumber, token string) error
}

type Issue struct {
	Repo                string      `json:"url"`
	Title               string      `json:"title"`
	Description         string      `json:"body"`
	IssueNumber         json.Number `json:"number,omitempty"` //TODO change here and everywhere to int and check it's working
	State               string      `json:"state,omitempty"`
	LastUpdateTimestamp string      `json:"updated_at"`
}
