package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
)

type fakeExecutor struct {
	mu      sync.Mutex
	replies []processResult
	calls   [][]string
	handle  func(context.Context, []string) processResult
}

func (f *fakeExecutor) Execute(ctx context.Context, args ...string) processResult {
	if e := ctx.Err(); e != nil {
		return processResult{Err: e, ExitCode: -1}
	}
	f.mu.Lock()
	f.calls = append(f.calls, append([]string(nil), args...))
	h := f.handle
	if h != nil {
		f.mu.Unlock()
		return h(ctx, args)
	}
	if len(f.replies) == 0 {
		f.mu.Unlock()
		return processResult{Err: errors.New("unexpected fake invocation"), ExitCode: 1}
	}
	r := f.replies[0]
	f.replies = f.replies[1:]
	f.mu.Unlock()
	return r
}
func TestMain(m *testing.M) {
	if os.Getenv("PR_REPORT_EXECUTOR_HELPER") == "1" {
		for _, a := range os.Args {
			if a == "wait" {
				time.Sleep(time.Minute)
			}
		}
		b, _ := json.Marshal(os.Args[1:])
		fmt.Println(string(b))
		fmt.Fprint(os.Stderr, os.Getenv("PR_REPORT_TEST_ENV"))
		os.Exit(7)
	}
	os.Exit(m.Run())
}
func TestGHExecutorArgumentsAndStreams(t *testing.T) {
	t.Setenv("PR_REPORT_EXECUTOR_HELPER", "1")
	t.Setenv("PR_REPORT_TEST_ENV", "inherited")
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	r := (ghExecutor{exe}).Execute(context.Background(), "-f", "query=$(touch /tmp/never); `echo nope`", "-F", "owner=org")
	var args []string
	if e = json.Unmarshal(r.Stdout, &args); e != nil {
		t.Fatal(e)
	}
	want := []string{"api", "graphql", "--hostname", "github.com", "-f", "query=$(touch /tmp/never); `echo nope`", "-F", "owner=org"}
	if !reflect.DeepEqual(args, want) || string(r.Stderr) != "inherited" || r.ExitCode != 7 || r.Err == nil {
		t.Fatalf("result %+v args %v", r, args)
	}
}
func TestExecutorCancellation(t *testing.T) {
	t.Setenv("PR_REPORT_EXECUTOR_HELPER", "1")
	exe, _ := os.Executable()
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	r := (ghExecutor{exe}).Execute(ctx, "wait")
	if !errors.Is(r.Err, context.DeadlineExceeded) {
		t.Fatal(r.Err)
	}
	f := &fakeExecutor{}
	r = f.Execute(ctx)
	if !errors.Is(r.Err, context.DeadlineExceeded) || len(f.calls) != 0 {
		t.Fatal(r)
	}
}
func TestLocateGHMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, e := locateGH(); e == nil {
		t.Fatal("found gh in empty path")
	}
}
