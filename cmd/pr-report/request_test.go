package main

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

func TestPerAttemptTimeout(t *testing.T) {
	f := &fakeExecutor{handle: func(ctx context.Context, _ []string) processResult {
		<-ctx.Done()
		return processResult{Err: ctx.Err(), ExitCode: -1}
	}}
	c := newClient(f)
	c.timeout = 5 * time.Millisecond
	r := c.collect(context.Background(), config{repos: []string{"org/one", "org/two"}})
	if len(f.calls) != 2 || len(r.Errors) != 2 || r.Complete {
		t.Fatal(r)
	}
	for _, e := range r.Errors {
		if e.Code != "timeout" {
			t.Fatal(e)
		}
	}
}
func TestCancellationStopsExecutionAndPendingWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := &fakeExecutor{handle: func(ctx context.Context, _ []string) processResult {
		cancel()
		<-ctx.Done()
		return processResult{Err: ctx.Err(), ExitCode: -1}
	}}
	var out, diag bytes.Buffer
	code := runReport(ctx, config{repos: []string{"org/one", "org/two"}, format: "json"}, &out, &diag, newClient(f))
	if code != 130 || len(f.calls) != 1 || out.Len() != 0 {
		t.Fatal(code, len(f.calls), out.String())
	}
	r := newClient(f).attempt(ctx)
	if !errors.Is(r.Err, context.Canceled) || len(f.calls) != 1 {
		t.Fatal(r)
	}
}
