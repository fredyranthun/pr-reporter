package main

import (
	"context"
	"testing"
)

func TestCommentIndicator(t *testing.T) {
	for _, a := range []*int{nil, ptr(0), ptr(2)} {
		for _, b := range []*int{nil, ptr(0), ptr(3)} {
			got := commentIndicator(a, b)
			positive := a != nil && *a > 0 || b != nil && *b > 0
			if positive {
				if got == nil || !*got {
					t.Fatal(a, b, got)
				}
			} else if a != nil && b != nil {
				if got == nil || *got {
					t.Fatal(a, b, got)
				}
			} else if got != nil {
				t.Fatal(a, b, got)
			}
		}
	}
}
func TestReviewBodyOnlyAndResolvedThreads(t *testing.T) {
	for _, threads := range []int{0, 1} {
		p := prFixture(1)
		p["reviews"] = map[string]any{"nodes": []any{map[string]any{"body": "Review text only"}}}
		p["reviewThreads"] = map[string]any{"totalCount": threads}
		f := &fakeExecutor{replies: []processResult{listFixture([]any{p}, false, nil)}}
		if threads > 0 {
			f.replies = append(f.replies, threadFixture([]any{threadNode("resolved", true)}, false, nil))
		}
		r := newClient(f).collect(context.Background(), config{repos: []string{"org/repo"}})
		if !r.Complete || r.PullRequests[0].HasComments == nil || *r.PullRequests[0].HasComments != (threads > 0) || *r.PullRequests[0].UnresolvedThreadsCount != 0 {
			t.Fatal(r)
		}
	}
	// A later unresolved thread cannot fill an unavailable principal count.
	p := prFixture(1)
	delete(p, "reviewThreads")
	f := &fakeExecutor{replies: []processResult{listFixture([]any{p}, false, nil), threadFixture([]any{threadNode("new", false)}, false, nil)}}
	r := newClient(f).collect(context.Background(), config{repos: []string{"org/repo"}})
	if r.PullRequests[0].HasComments != nil || *r.PullRequests[0].UnresolvedThreadsCount != 1 {
		t.Fatal(r)
	}
}
