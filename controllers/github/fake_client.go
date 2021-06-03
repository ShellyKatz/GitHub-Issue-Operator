package github

import (
	"encoding/json"
	"fmt"
	examplev1alpha1 "github.com/ShellyKatz/example-operator/api/v1alpha1"
	"time"
)

//TODO http test (pkg), mockgen (mock generator)
type FakeClient struct {
	Issues []Issue
	Err error
}

func NewFakeClient(issues []Issue) *FakeClient {
	return &FakeClient{
		Issues: issues,
	}
}

func (f *FakeClient) FindIssue(ghIssueSpec examplev1alpha1.GitHubIssueSpec, token string) (*Issue, error) {
	//check if there's an item in the repository issues list with the matching title
	for _, issue := range f.Issues {
		if ghIssueSpec.Title == issue.Title {
			return &issue, nil
		}
	}

	//if there's no item in the repository issues list with the matching title
	return nil, fmt.Errorf(TitleNotFound)
}

func (f *FakeClient) Create(ghIssueSpec examplev1alpha1.GitHubIssueSpec, token string) (*Issue, error) {
	if f.Err != nil {
		return &Issue{}, f.Err
	}
	issue := Issue{
		Repo: "https://api.github.com/repos" +ghIssueSpec.Repo + "/issues",
		Title: ghIssueSpec.Title,
		Description: ghIssueSpec.Description,
		IssueNumber: json.Number(len(f.Issues)+1),
		State: "open",
		LastUpdateTimestamp: time.Now().String(), //"2021-05-31T07:49:28Z",
	}
	f.Issues = append(f.Issues, issue)
	return &issue, nil
}

func (f *FakeClient) Edit(ghIssueSpec examplev1alpha1.GitHubIssueSpec, issueNumber, token string) error {
	panic("implement me")
}

func (f *FakeClient) Close(ghIssueSpec examplev1alpha1.GitHubIssueSpec, issueNumber, token string) error {
	panic("implement me")
}

func (f *FakeClient) FailsOnCreate() {
	f.Err = fmt.Errorf("always failing client")
}