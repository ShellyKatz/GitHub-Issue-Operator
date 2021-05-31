package github

import examplev1alpha1 "github.com/ShellyKatz/example-operator/api/v1alpha1"

//TODO http test (pkg), mockgen (mock generator)
type fakeClient struct {
	issues []Issue
}

func newFakeClient(issues []Issue) fakeClient {
	return fakeClient{
		issues: issues,
	}
}

func (f *fakeClient) FindIssue(ghIssueSpec examplev1alpha1.GitHubIssueSpec, token string) (*Issue, error) {
	panic("implement me")
}

func (f *fakeClient) Create(ghIssueSpec examplev1alpha1.GitHubIssueSpec, token string) (*Issue, error) {
	panic("implement me")
}

func (f *fakeClient) Edit(ghIssueSpec examplev1alpha1.GitHubIssueSpec, issueNumber, token string) error {
	panic("implement me")
}

func (f *fakeClient) Close(ghIssueSpec examplev1alpha1.GitHubIssueSpec, issueNumber, token string) error {
	panic("implement me")
}
