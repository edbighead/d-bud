package main

type Response struct {
	Values []PullRequest
}

type PullRequest struct {
	Title string
	State string
	ToRef struct {
		DisplayID string
	}
	FromRef struct {
		Repository struct {
			Slug string
		}
	}
	Issues []string
}

type Issue struct {
	IssueID     string
	Pullrequest []PullRequest
}
