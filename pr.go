package main

type PullRequest struct {
	Values []struct {
		Title string
		State string
		ToRef struct {
			DisplayID string
		}
	}
}
