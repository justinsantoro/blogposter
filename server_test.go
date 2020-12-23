package main

import (
	"testing"
)

func TestPostnameFromURL(t *testing.T) {
	urls := []string{
		"https://localhost:1313/post/test-post",
		"https://localhost:1313/post/test-post/",
		"https://localhost:1313/post/test-post?test=true",
	}
	for _, url := range urls {
		if post := PostnameFromURL(url); post != "test-post" {
			t.Errorf("did not correctly parse out post name from %s: got [%s]", url, post)
		}
	}
}
