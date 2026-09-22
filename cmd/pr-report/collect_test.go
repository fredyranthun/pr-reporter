package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCollectMultipleRepositories(t *testing.T) {
	third := listFixture([]any{prFixture(1)}, false, nil)
	third.Stdout = []byte(strings.ReplaceAll(string(third.Stdout), "Org/Repo", "Other/Repo"))
	f := &fakeExecutor{replies: []processResult{listFixture([]any{prFixture(1)}, false, nil), {Stdout: []byte(`{"data":{"repository":null}}`)}, third, listFixture([]any{}, false, nil)}}
	c := newClient(f)
	ticks := int64(0)
	c.now = func() time.Time { ticks++; return time.Unix(ticks, 0).In(time.FixedZone("other", 3600)) }
	r := c.collect(context.Background(), config{repos: []string{"ORG/repo", "org/private", "other/repo", "org/empty"}})
	if r.Complete || len(r.Repositories) != 4 || len(r.PullRequests) != 2 || r.PRsCollected != 2 || len(r.Errors) != 1 {
		t.Fatalf("%+v", r)
	}
	if r.Repositories[0].RequestedRepo != "ORG/repo" || *r.Repositories[0].Repo != "Org/Repo" || r.Repositories[1].Repo != nil || r.Repositories[1].ListingComplete || !r.Repositories[3].ListingComplete {
		t.Fatal(r.Repositories)
	}
	if r.PullRequests[0].Number != r.PullRequests[1].Number || r.PullRequests[0].Repo == r.PullRequests[1].Repo {
		t.Fatal(r.PullRequests)
	}
	if r.StartedAt.Location() != time.UTC || r.FinishedAt.Location() != time.UTC || !r.FinishedAt.After(r.PullRequests[1].ObservedAt) || !r.PullRequests[0].ObservedAt.After(r.StartedAt) {
		t.Fatal(r)
	}
	for i, repo := range []string{"name=repo", "name=private", "name=repo", "name=empty"} {
		if !strings.Contains(strings.Join(f.calls[i], " "), repo) {
			t.Fatal(f.calls)
		}
	}
}

func TestCompletenessAndCounters(t *testing.T) {
	for _, scenario := range []string{"empty", "all fail", "unknown", "contradiction", "nullable", "missing detail", "partial listing complete details", "partial listing missing details", "duplicate incomplete wins"} {
		t.Run(scenario, func(t *testing.T) {
			p := prFixture(1)
			f := &fakeExecutor{}
			wantComplete := true
			wantListing := true
			wantDetails := true
			wantCount := 1
			wantErrors := 0
			switch scenario {
			case "empty":
				wantCount = 0
				f.replies = []processResult{listFixture([]any{}, false, nil)}
			case "all fail":
				wantCount = 0
				wantComplete = false
				wantListing = false
				wantErrors = 1
				f.replies = []processResult{{Stdout: []byte(`{"data":{"repository":null}}`)}}
			case "unknown":
				p["mergeStateStatus"] = "UNKNOWN"
			case "contradiction":
				p["isDraft"] = true
			case "missing detail":
				delete(p, "comments")
				wantComplete = false
				wantDetails = false
				wantErrors = 1
			case "partial listing complete details", "partial listing missing details":
				wantComplete = false
				wantListing = false
				wantErrors = 1
				if scenario == "partial listing missing details" {
					delete(p, "author")
					wantDetails = false
					wantErrors++
				}
				f.replies = []processResult{listFixture([]any{p}, true, "a"), {Stdout: []byte(`{}`)}}
			case "duplicate incomplete wins":
				delete(p, "author")
				duplicate := prFixture(1)
				duplicate["mergeable"] = "FUTURE"
				f.replies = []processResult{listFixture([]any{p}, true, "a"), listFixture([]any{duplicate}, false, nil)}
				wantComplete = false
				wantDetails = false
				wantErrors = 1
			}
			if len(f.replies) == 0 {
				f.replies = []processResult{listFixture([]any{p}, false, nil)}
			}
			r := newClient(f).collect(context.Background(), config{repos: []string{"Org/repo"}})
			if r.Complete != wantComplete || r.Repositories[0].ListingComplete != wantListing || r.Repositories[0].DetailsComplete != wantDetails || r.PRsCollected != wantCount || r.PRsReturned != wantCount || r.Repositories[0].PRsCollected != wantCount || len(r.Errors) != wantErrors || len(r.Warnings) != 0 {
				t.Fatalf("%+v", r)
			}
			if (r.Repositories[0].Repo == nil) != (scenario == "all fail") {
				t.Fatal(r.Repositories)
			}
		})
	}
}
