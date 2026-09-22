package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func argument(args []string, key string) string {
	for _, a := range args {
		if strings.HasPrefix(a, key+"=") {
			return strings.TrimPrefix(a, key+"=")
		}
	}
	return ""
}
func TestConcurrentCollectionStableOutput(t *testing.T) {
	var mu sync.Mutex
	active, peak := 0, 0
	f := &fakeExecutor{handle: func(ctx context.Context, args []string) processResult {
		mu.Lock()
		active++
		if active > peak {
			peak = active
		}
		mu.Unlock()
		defer func() { mu.Lock(); active--; mu.Unlock() }()
		if err := waitContext(ctx, 2*time.Millisecond); err != nil {
			return processResult{Err: err, ExitCode: -1}
		}
		if argument(args, "query") == threadQuery {
			return threadFixture([]any{threadNode("x", false)}, false, nil)
		}
		nodes := []any{}
		for n := 3; n >= 1; n-- {
			p := prFixture(n)
			p["reviewThreads"] = map[string]any{"totalCount": 1}
			nodes = append(nodes, p)
		}
		r := listFixture(nodes, false, nil)
		r.Stdout = []byte(strings.ReplaceAll(string(r.Stdout), "Org/Repo", "Org/"+argument(args, "name")))
		return r
	}}
	r := newClient(f).collect(context.Background(), config{repos: []string{"org/c", "org/b", "org/a"}, concurrency: 3})
	if !r.Complete || len(r.PullRequests) != 9 || peak < 2 || peak > 3 || active != 0 || len(f.calls) != 12 {
		t.Fatal(r, peak, active, len(f.calls))
	}
	for i, p := range r.PullRequests {
		if p.Repo != fmt.Sprint("Org/", string(rune('a'+i/3))) || p.Number != i%3+1 || *p.UnresolvedThreadsCount != 1 {
			t.Fatal(r.PullRequests)
		}
	}
	if r.Repositories[0].RequestedRepo != "org/c" {
		t.Fatal("repository order changed")
	}
}
