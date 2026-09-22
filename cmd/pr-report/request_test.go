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
	c.wait = func(context.Context, time.Duration) error { return nil }
	r := c.collect(context.Background(), config{repos: []string{"org/one", "org/two"}})
	if len(f.calls) != 6 || len(r.Errors) != 2 || r.Complete {
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

func TestRetryBudgetAndBackoff(t *testing.T) {
	for _, failures := range []int{1, 2, 3} {
		f := &fakeExecutor{}
		for i := 0; i < failures; i++ {
			f.replies = append(f.replies, processResult{Err: errors.New("failed"), ExitCode: 1, Stderr: []byte("HTTP 502")})
		}
		f.replies = append(f.replies, listFixture([]any{}, false, nil))
		c := newClient(f)
		var waits []time.Duration
		c.wait = func(_ context.Context, d time.Duration) error { waits = append(waits, d); return nil }
		c.jitter = func() time.Duration { return 20 * time.Millisecond }
		r := c.collect(context.Background(), config{repos: []string{"org/repo"}})
		wantCalls := failures + 1
		if wantCalls > 3 {
			wantCalls = 3
		}
		if len(f.calls) != wantCalls || r.Complete != (failures < 3) || len(waits) != wantCalls-1 {
			t.Fatal(r, len(f.calls), waits)
		}
		for i, d := range waits {
			if d != time.Duration(i+1)*time.Second+20*time.Millisecond {
				t.Fatal(waits)
			}
		}
	}
}
func TestRetryClassification(t *testing.T) {
	for _, tc := range []struct {
		reply processResult
		retry bool
	}{
		{processResult{Err: context.DeadlineExceeded, ExitCode: -1}, true},
		{processResult{ExitCode: 1, Stderr: []byte("dial: no such host")}, true},
		{processResult{ExitCode: 1, Stderr: []byte("HTTP 503")}, true},
		{processResult{ExitCode: 1, Stderr: []byte("HTTP 401")}, false},
		{processResult{ExitCode: 1, Stderr: []byte("HTTP 403")}, false},
		{processResult{ExitCode: 1, Stderr: []byte("HTTP 429")}, false},
		{processResult{Stdout: []byte(`{"errors":[{"message":"rate limited"}]}`)}, false},
		{processResult{Stdout: []byte(`{"errors":[{"message":"invalid query"}]}`)}, false},
		{processResult{Stdout: []byte(`{}`)}, false},
	} {
		f := &fakeExecutor{replies: []processResult{tc.reply, listFixture([]any{}, false, nil)}}
		c := newClient(f)
		c.wait = func(context.Context, time.Duration) error { return nil }
		c.request(context.Background())
		want := 1
		if tc.retry {
			want = 2
		}
		if len(f.calls) != want {
			t.Fatal(tc, len(f.calls))
		}
	}
}
func TestCancelDuringRetryWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := &fakeExecutor{replies: []processResult{{Err: context.DeadlineExceeded, ExitCode: -1}}}
	c := newClient(f)
	c.wait = func(ctx context.Context, _ time.Duration) error { cancel(); return waitContext(ctx, time.Hour) }
	r := c.request(ctx)
	if !errors.Is(r.Err, context.Canceled) || len(f.calls) != 1 {
		t.Fatal(r)
	}
	for i := 0; i < 20; i++ {
		if j := retryJitter(); j < 0 || j > 100*time.Millisecond {
			t.Fatal(j)
		}
	}
}
