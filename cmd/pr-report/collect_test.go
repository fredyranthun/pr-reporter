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
