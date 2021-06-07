package github

import (
	"encoding/json"
	"fmt"
	examplev1alpha1 "github.com/ShellyKatz/example-operator/api/v1alpha1"
	"strconv"
)

const CreatError = "client fails on create"
const EditError = "client fails on Edit"
const DeleteError = "client fails on Delete"
const StatusUpdateError = "client fails "

//TODO http test (pkg), mockgen (mock generator)
type FakeClient struct {
	Issues []*Issue
	Err    error
}

func NewFakeClient(issues []*Issue, fails bool, message string) *FakeClient {
	if fails == true {
		return &FakeClient{
			Issues: issues,
			Err:    fmt.Errorf(message),
		}
	}
	return &FakeClient{
		Issues: issues,
	}
}

func (f *FakeClient) FindIssue(ghIssueSpec examplev1alpha1.GitHubIssueSpec, token string) (*Issue, error) {
	//check if there's an item in the repository issues list with the matching title
	for _, issue := range f.Issues {
		if ghIssueSpec.Title == issue.Title {
			return issue, nil
		}
	}

	//if there's no item in the repository issues list with the matching title
	return nil, fmt.Errorf(TitleNotFound)
}

func (f *FakeClient) Create(ghIssueSpec examplev1alpha1.GitHubIssueSpec, token string) (*Issue, error) {
	if fmt.Sprintf("%v", f.Err) == CreatError {
		return &Issue{}, f.Err
	}
	issue := Issue{
		Repo:                "https://api.github.com/repos" + ghIssueSpec.Repo + "/issues",
		Title:               ghIssueSpec.Title,
		Description:         ghIssueSpec.Description,
		IssueNumber:         json.Number(strconv.Itoa(len(f.Issues) + 1)),
		State:               "open",
		LastUpdateTimestamp: "2021-05-31T07:49:28Z",//time.Now().String(), //"2021-05-31T07:49:28Z",
	}
	f.Issues = append(f.Issues, &issue)
	return &issue, nil
}

func (f *FakeClient) Edit(ghIssueSpec examplev1alpha1.GitHubIssueSpec, issueNumber, token string) error {
	if fmt.Sprintf("%v", f.Err) == EditError {
		return f.Err
	}
	for _, issue := range f.Issues {
		if issue.Title == ghIssueSpec.Title {
			issue.Description = ghIssueSpec.Description
			return nil
		}
	}
	return fmt.Errorf("couldn't find issue title in repo")
}

func (f *FakeClient) Close(ghIssueSpec examplev1alpha1.GitHubIssueSpec, issueNumber, token string) error {
	if fmt.Sprintf("%v", f.Err) == DeleteError {
		return f.Err
	}
	for _, issue := range f.Issues {
		if issue.Title == ghIssueSpec.Title {
			issue.State = "closed"
			//f.Issues[i] = f.Issues[len(f.Issues)-1]
			//f.Issues = f.Issues[:len(f.Issues)-1]
			return nil
		}
	}
	return fmt.Errorf("couldn't find issue title in repo")
}

//func (f *FakeClient) Fail(message string) {
//	f.Err = fmt.Errorf(message)
//}
