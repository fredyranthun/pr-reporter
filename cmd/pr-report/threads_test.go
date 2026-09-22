package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func threadFixture(nodes []any, next bool, cursor any) processResult {
	return response(map[string]any{"data": map[string]any{"repository": map[string]any{"pullRequest": map[string]any{"reviewThreads": map[string]any{"nodes": nodes, "pageInfo": map[string]any{"hasNextPage": next, "endCursor": cursor}}}}}})
}
func threadNode(id string, resolved bool) map[string]any {
	return map[string]any{"id": id, "isResolved": resolved, "isOutdated": true}
}
func TestThreads130AndDuplicates(t *testing.T) {
	first, second := []any{}, []any{}
	for i := 0; i < 130; i++ {
		n := threadNode(fmt.Sprint(i), i%2 == 0)
		if i < 100 {
			first = append(first, n)
		} else {
			second = append(second, n)
		}
	}
	first = append(first, threadNode("1", true))
	second = append(second, threadNode("0", false), threadNode("1", true))
	f := &fakeExecutor{replies: []processResult{threadFixture(first, true, "thread-a"), threadFixture(second, false, nil)}}
	n, e := newClient(f).countThreads(context.Background(), pullRequest{Repo: "Org/Repo", Number: 5, ReviewThreadsCount: ptr(130)})
	if e != nil || n == nil || *n != 65 || len(f.calls) != 2 {
		t.Fatal(n, e)
	}
	if !strings.Contains(strings.Join(f.calls[1], " "), "cursor=thread-a") || !strings.Contains(strings.Join(f.calls[0], " "), "number=5") || strings.Contains(threadQuery, "body") {
		t.Fatal(f.calls)
	}
}
func TestThreadSkipIndependentCursorsAndTemporalCounts(t *testing.T) {
	p1, p2, p3 := prFixture(1), prFixture(2), prFixture(3)
	p1["reviewThreads"] = map[string]any{"totalCount": 1}
	delete(p2, "reviewThreads")
	f := &fakeExecutor{replies: []processResult{listFixture([]any{p1, p1, p2, p3}, false, nil), threadFixture([]any{threadNode("a", false)}, true, "a"), threadFixture([]any{threadNode("b", false)}, false, nil), threadFixture([]any{threadNode("a", false)}, false, nil)}}
	r := newClient(f).collect(context.Background(), config{repos: []string{"org/repo"}})
	if len(f.calls) != 4 || len(r.PullRequests) != 3 || *r.PullRequests[0].ReviewThreadsCount != 1 || *r.PullRequests[0].UnresolvedThreadsCount != 2 || r.PullRequests[1].ReviewThreadsCount != nil || *r.PullRequests[1].UnresolvedThreadsCount != 1 || r.PullRequests[1].DetailsComplete || *r.PullRequests[2].UnresolvedThreadsCount != 0 {
		t.Fatal(r)
	}
	for _, arg := range f.calls[3] {
		if strings.HasPrefix(arg, "cursor=") {
			t.Fatal("cursor leaked", f.calls)
		}
	}
	f = &fakeExecutor{replies: []processResult{threadFixture([]any{}, false, nil)}}
	n, e := newClient(f).countThreads(context.Background(), pullRequest{ReviewThreadsCount: ptr(2)})
	if e != nil || *n != 0 {
		t.Fatal(n, e)
	}
}

func TestThreadFailurePreservesPRWithoutSubtotal(t *testing.T) {
	for _, bad := range []processResult{
		{Stdout: []byte(`{"data":{"repository":{"pullRequest":null}}}`)},
		{Stdout: []byte(`{"data":{"repository":null},"errors":[{"type":"NOT_FOUND"}]}`)},
		{Stdout: []byte(`{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[],"pageInfo":{"hasNextPage":false}}}}}},"errors":[{"message":"partial"}]}`)},
		threadFixture([]any{threadNode("new", false)}, true, "a"),
		threadFixture([]any{threadNode("new", false)}, true, nil),
		threadFixture([]any{map[string]any{"id": "bad"}}, false, nil),
	} {
		p := prFixture(1)
		p["reviewThreads"] = map[string]any{"totalCount": 3}
		f := &fakeExecutor{replies: []processResult{listFixture([]any{p}, false, nil), threadFixture([]any{threadNode("first", false)}, true, "a"), bad}}
		r := newClient(f).collect(context.Background(), config{repos: []string{"org/repo"}})
		if len(r.PullRequests) != 1 || r.PullRequests[0].UnresolvedThreadsCount != nil || r.PullRequests[0].DetailsComplete || r.Complete || !r.Repositories[0].ListingComplete || r.Repositories[0].DetailsComplete || len(r.Errors) != 1 {
			t.Fatalf("%+v", r)
		}
		pOut := r.PullRequests[0]
		if *pOut.ReviewThreadsCount != 3 || pOut.Title != "Title" || *pOut.Mergeable != "MERGEABLE" {
			t.Fatal(pOut)
		}
		e := r.Errors[0]
		if e.Stage != "review_threads" || *e.Repo != "Org/Repo" || *e.PRNumber != 1 {
			t.Fatal(e)
		}
	}
}
func TestThreadRequiredFields(t *testing.T) {
	for _, key := range []string{"id", "isResolved"} {
		for _, null := range []bool{false, true} {
			node := threadNode("x", false)
			if null {
				node[key] = nil
			} else {
				delete(node, key)
			}
			if _, e := decodeThreads(threadFixture([]any{node}, false, nil)); e == nil {
				t.Fatal(key, null)
			}
		}
	}
	for _, raw := range []string{`{}`, `{"errors":null}`, `{"errors":{}}`, `{"data":{"repository":{}}}`} {
		if _, e := decodeThreads(processResult{Stdout: []byte(raw)}); e == nil {
			t.Fatal(raw)
		}
	}
	good := threadFixture([]any{}, false, nil)
	good.ExitCode = 1
	if _, e := decodeThreads(good); e == nil {
		t.Fatal("failed process accepted")
	}
	for _, node := range []any{nil, map[string]any{"id": "", "isResolved": false}, map[string]any{"id": "x", "isResolved": "false"}} {
		if _, e := decodeThreads(threadFixture([]any{node}, false, nil)); e == nil {
			t.Fatal(node)
		}
	}
}
