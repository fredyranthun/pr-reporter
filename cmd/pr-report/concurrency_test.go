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

func TestConcurrencyBoundsRetriesAndThreads(t *testing.T) {
	for _, limit := range []int{1, 3, 8} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			var mu sync.Mutex
			active, peak := 0, 0
			attempts := map[string]int{}
			f := &fakeExecutor{handle: func(ctx context.Context, args []string) processResult {
				mu.Lock()
				active++
				if active > peak {
					peak = active
				}
				key := argument(args, "name") + argument(args, "number")
				attempts[key]++
				nth := attempts[key]
				mu.Unlock()
				defer func() { mu.Lock(); active--; mu.Unlock() }()
				if err := waitContext(ctx, 3*time.Millisecond); err != nil {
					return processResult{Err: err, ExitCode: -1}
				}
				if nth == 1 {
					return processResult{Err: context.DeadlineExceeded, ExitCode: -1}
				}
				if argument(args, "query") == threadQuery {
					return threadFixture([]any{threadNode("pending", false)}, false, nil)
				}
				p := prFixture(1)
				p["reviewThreads"] = map[string]any{"totalCount": 1}
				r := listFixture([]any{p}, false, nil)
				r.Stdout = []byte(strings.ReplaceAll(string(r.Stdout), "Org/Repo", "Org/"+argument(args, "name")))
				return r
			}}
			c := newClient(f)
			c.wait = func(ctx context.Context, _ time.Duration) error { return ctx.Err() }
			c.jitter = func() time.Duration { return 0 }
			started := time.Now()
			repos := []string{}
			for i := 0; i < 9; i++ {
				repos = append(repos, fmt.Sprintf("org/repo%d", i))
			}
			r := c.collect(context.Background(), config{repos: repos, concurrency: limit})
			duration := time.Since(started)
			t.Logf("concurrency=%d, 9 repositories, 9 thread queries: %s", limit, duration)
			if !r.Complete || r.PRsCollected != 9 || len(r.Errors) != 0 || len(f.calls) != 36 || peak > limit || active != 0 {
				t.Fatal(peak, active, len(f.calls), r)
			}
			if limit == 1 && peak != 1 || limit > 1 && peak < 2 {
				t.Fatal("limit unused", peak)
			}
		})
	}
}
func TestConcurrentPartialFailureAndCancellation(t *testing.T) {
	f := &fakeExecutor{handle: func(ctx context.Context, args []string) processResult {
		if argument(args, "name") == "bad" {
			return processResult{Stdout: []byte(`{"data":{"repository":null}}`)}
		}
		r := listFixture([]any{prFixture(1)}, false, nil)
		r.Stdout = []byte(strings.ReplaceAll(string(r.Stdout), "Org/Repo", "Org/"+argument(args, "name")))
		return r
	}}
	r := newClient(f).collect(context.Background(), config{repos: []string{"org/good", "org/bad", "org/other"}, concurrency: 3})
	if r.Complete || len(r.Errors) != 1 || r.PRsCollected != 2 || len(r.Repositories) != 3 || r.Repositories[1].Repo != nil {
		t.Fatal(r)
	}
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	var once sync.Once
	f = &fakeExecutor{handle: func(ctx context.Context, _ []string) processResult {
		once.Do(func() { close(started) })
		<-ctx.Done()
		return processResult{Err: ctx.Err(), ExitCode: -1}
	}}
	done := make(chan report, 1)
	go func() {
		done <- newClient(f).collect(ctx, config{repos: []string{"org/a", "org/b", "org/c", "org/d"}, concurrency: 3})
	}()
	<-started
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancellation deadlocked")
	}
	if len(f.calls) > 3 {
		t.Fatal("started queued work after cancel", len(f.calls))
	}
}
