package main

import "reflect"

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
	Links struct {
		Self []struct {
			Href string
		}
	}
	Issues []string
}

type IssuePR struct {
	Issue       Issue
	Pullrequest []PullRequest
}

type queryObject struct {
	Branch string
	JQL    string
}

type JiraResponse struct {
	Issues []Issue
}

type Issue struct {
	Key    string
	Fields struct {
		Summary string
		Project struct {
			Key string
		}
	}
}

func AppendIfMissingPullRequest(slice []PullRequest, i PullRequest) []PullRequest {
	for _, ele := range slice {
		if reflect.DeepEqual(ele, i) {
			return slice
		}
	}
	return append(slice, i)
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func AppendIfMissing(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

func difference(a, b []string) []string {
	mb := map[string]bool{}
	for _, x := range b {
		mb[x] = true
	}
	ab := []string{}
	for _, x := range a {
		if _, ok := mb[x]; !ok {
			ab = append(ab, x)
		}
	}
	return ab
}

func (box *IssuePR) addItem(item PullRequest) []PullRequest {
	box.Pullrequest = append(box.Pullrequest, item)
	return box.Pullrequest
}
