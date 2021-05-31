package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	examplev1alpha1 "github.com/ShellyKatz/example-operator/api/v1alpha1"
	"io/ioutil"
	"net/http"
	"strings"
)

const UrlPrefix = "https://api.github.com/repos/"
const TitleNotFound = "object title not found on github"

type ClientAPI struct {
	httpClient http.Client
	token      string
}

//func NewGithubClient() ClientAPI {
//	return ClientAPI{
//		httpClient: http.Client{},
//		token:      os.Getenv("GITHUB_TOKEN"),
//	}
//}

// NewIssue https://vorozhko.net/create-github-issue-ticket-with-golang
// specify data fields for new github issue submission

type NewIssue struct {
	Title       string `json:"title"`
	Description string `json:"body"`
}

type Repo struct {
	Repo  string `json:"repo"`
	Owner string `json:"owner"`
}

func (c *ClientAPI) FindIssue(ghIssueSpec examplev1alpha1.GitHubIssueSpec, token string) (*Issue, error) {
	apiURL := "https://api.github.com/repos/" + ghIssueSpec.Repo + "/issues?state=all"
	// split ownerRepo to owner and repository
	ownerAndRepo := strings.Split(ghIssueSpec.Repo, "/")
	// title is the only required field
	repoData := Repo{Repo: ownerAndRepo[0], Owner: ownerAndRepo[1]}
	// make it json
	jsonData, _ := json.Marshal(repoData)
	// creating client to set custom headers for Authorization
	req, _ := http.NewRequest("GET", apiURL, bytes.NewReader(jsonData))
	req.Header.Set("Authorization", "token "+token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	// print body as it may contain hints in case of errors
	//fmt.Println(string(body))

	var issues []Issue
	err = json.Unmarshal(body, &issues)

	if err != nil {
		return nil, err
	}

	// loop over issues titles and look for the title given to the function
	for _, issue := range issues {
		if issue.Title == ghIssueSpec.Title {
			fmt.Printf("!!!!!1repo: %s", issue.Repo)
			return &issue, nil
		}
	}
	err = fmt.Errorf(TitleNotFound)

	return nil, err
}

// function I copied from:
// https://vorozhko.net/create-github-issue-ticket-with-golang
func (c *ClientAPI) Create(ghIssueSpec examplev1alpha1.GitHubIssueSpec, token string) (*Issue, error) {
	apiURL := "https://api.github.com/repos/" + ghIssueSpec.Repo + "/issues"
	// title is the only required field
	issueData := NewIssue{Title: ghIssueSpec.Title, Description: ghIssueSpec.Description}
	// make it json
	jsonData, _ := json.Marshal(issueData)
	// set custom headers for Authorization
	req, _ := http.NewRequest("POST", apiURL, bytes.NewReader(jsonData))

	req.Header.Set("Authorization", "token "+token)
	resp, err := c.httpClient.Do(req)
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

func (c *ClientAPI) Edit(ghIssueSpec examplev1alpha1.GitHubIssueSpec, issueNumber, token string) error {
	apiURL := "https://api.github.com/repos/" + ghIssueSpec.Repo + "/issues/" + issueNumber
	// title is the only required field
	issueData := Issue{Repo: ghIssueSpec.Repo, Title: ghIssueSpec.Title, Description: ghIssueSpec.Description,
		IssueNumber: json.Number(issueNumber)}
	// make it json
	jsonData, _ := json.Marshal(issueData)
	// creating client to set custom headers for Authorization
	req, _ := http.NewRequest("PATCH", apiURL, bytes.NewReader(jsonData))
	req.Header.Set("Authorization", "token "+token)
	resp, err := c.httpClient.Do(req)
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

// Close : close github issue
func (c *ClientAPI) Close(ghIssueSpec examplev1alpha1.GitHubIssueSpec, issueNumber, token string) error {
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
