package main

import (
	"context"
	"testing"
)

func TestSortingAndPostCollectionFilter(t *testing.T) {
	r := report{Filters: reportFilters{true}, PullRequests: list[pullRequest]{{Repo: "z/repo", Number: 1, UnresolvedThreadsCount: ptr(1)}, {Repo: "B/repo", Number: 2, UnresolvedThreadsCount: ptr(2)}, {Repo: "b/Repo", Number: 1, UnresolvedThreadsCount: ptr(3)}, {Repo: "a/repo", Number: 1}, {Repo: "a/repo", Number: 2, UnresolvedThreadsCount: ptr(0)}}}
	sortAndFilter(&r)
	if r.PRsCollected != 5 || r.PRsReturned != 3 || r.PullRequests[0].Number != 1 || r.PullRequests[1].Number != 2 || r.PullRequests[2].Repo != "z/repo" {
		t.Fatal(r)
	}
	for _, allFiltered := range []bool{false, true} {
		p1, p2, p3 := prFixture(1), prFixture(2), prFixture(3)
		p2["reviewThreads"] = map[string]any{"totalCount": 1}
		p3["reviewThreads"] = map[string]any{"totalCount": 1}
		threads := []any{}
		if !allFiltered {
			threads = append(threads, threadNode("one", false))
		}
		f := &fakeExecutor{replies: []processResult{listFixture([]any{p3, p1, p2}, false, nil), {Stdout: []byte(`{}`)}, threadFixture(threads, false, nil)}}
		r := newClient(f).collect(context.Background(), config{repos: []string{"org/repo"}, onlyUnresolved: true})
		want := 1
		if allFiltered {
			want = 0
		}
		if r.Complete || !r.Filters.OnlyUnresolved || r.PRsCollected != 3 || r.PRsReturned != want || r.Repositories[0].PRsCollected != 3 || r.Repositories[0].DetailsComplete || len(r.Errors) != 1 || *r.Errors[0].PRNumber != 3 {
			t.Fatal(r)
		}
	}
}
