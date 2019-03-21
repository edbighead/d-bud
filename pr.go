package main

type PR struct {
	Values []struct {
		Title string
		State string
		ToRef struct {
			DisplayID string
		}
	}
}
