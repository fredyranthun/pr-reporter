package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestList205PRs(t *testing.T) {
	f := &fakeExecutor{}
	for start := 1; start <= 205; start += 100 {
		nodes := []any{}
		for n := start; n < start+100 && n <= 205; n++ {
			nodes = append(nodes, prFixture(n))
		}
		f.replies = append(f.replies, listFixture(nodes, start < 201, fmt.Sprint(start)))
	}
	c := newClient(f)
	c.now = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	r, prs, errs := c.listPRs(context.Background(), "org/repo")
	if !r.ListingComplete || len(prs) != 205 || len(errs) != 0 || len(f.calls) != 3 {
		t.Fatalf("%+v %d %v", r, len(prs), errs)
	}
	if !strings.Contains(strings.Join(f.calls[1], " "), "cursor=1") || !strings.Contains(strings.Join(f.calls[2], " "), "cursor=101") {
		t.Fatal(f.calls)
	}
	for _, p := range prs {
		if !p.public.ObservedAt.Equal(c.now()) {
			t.Fatal(p)
		}
	}
}
func TestListDuplicateFirstWins(t *testing.T) {
	first := prFixture(1)
	delete(first, "author")
	duplicate := prFixture(1)
	duplicate["title"] = "later"
	duplicate["mergeable"] = "NEW"
	f := &fakeExecutor{replies: []processResult{listFixture([]any{first, first}, true, "a"), listFixture([]any{duplicate, prFixture(2)}, false, nil)}}
	c := newClient(f)
	tick := 0
	c.now = func() time.Time { tick++; return time.Unix(int64(tick), 0) }
	r, p, e := c.listPRs(context.Background(), "org/repo")
	if !r.ListingComplete || r.DetailsComplete || len(p) != 2 || len(e) != 1 || p[0].public.Title != "Title" || p[0].public.Author != nil || p[0].public.ObservedAt.Unix() != 1 || p[1].public.ObservedAt.Unix() != 2 {
		t.Fatalf("%+v %+v %v", r, p, e)
	}
}
func TestListRejectsWholeConflictingPage(t *testing.T) {
	for _, kind := range []string{"id", "number", "canonical", "malformed", "cursor", "missing_cursor", "failed"} {
		t.Run(kind, func(t *testing.T) {
			p := prFixture(1)
			second := listFixture([]any{prFixture(2), p}, false, nil)
			switch kind {
			case "id":
				p["number"] = 3
				second = listFixture([]any{prFixture(2), p}, false, nil)
			case "number":
				p["id"] = "different"
				second = listFixture([]any{prFixture(2), p}, false, nil)
			case "canonical":
				second.Stdout = []byte(strings.ReplaceAll(string(second.Stdout), `"Org/Repo"`, `"Other/Repo"`))
			case "malformed":
				delete(p, "isDraft")
				second = listFixture([]any{prFixture(2), p}, false, nil)
			case "cursor":
				second = listFixture([]any{prFixture(2)}, true, "a")
			case "missing_cursor":
				second = listFixture([]any{prFixture(2)}, true, nil)
			case "failed":
				second = processResult{Stdout: []byte(`{"errors":[{"message":"oops"}]}`)}
			}
			f := &fakeExecutor{replies: []processResult{listFixture([]any{prFixture(1)}, true, "a"), second}}
			r, prs, errs := newClient(f).listPRs(context.Background(), "org/repo")
			if r.ListingComplete || len(prs) != 1 || len(errs) != 1 || r.Repo == nil {
				t.Fatalf("%+v %d %v", r, len(prs), errs)
			}
		})
	}
	// Conflicts within the first page also invalidate all of it.
	p := prFixture(1)
	p["id"] = "another"
	f := &fakeExecutor{replies: []processResult{listFixture([]any{prFixture(1), p}, false, nil)}}
	r, prs, _ := newClient(f).listPRs(context.Background(), "org/repo")
	if r.Repo != nil || len(prs) != 0 {
		t.Fatal(r, prs)
	}
}
func TestEmptyPageThenFailure(t *testing.T) {
	f := &fakeExecutor{replies: []processResult{listFixture([]any{}, true, "a"), {Stdout: []byte(`{}`)}}}
	r, p, e := newClient(f).listPRs(context.Background(), "org/repo")
	if r.Repo == nil || r.ListingComplete || !r.DetailsComplete || len(p) != 0 || len(e) != 1 {
		t.Fatal(r, p, e)
	}
}
