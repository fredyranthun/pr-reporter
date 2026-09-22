package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func runFixture(t *testing.T, args []string, f *fakeExecutor) (int, report, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := run(args, &out, &errOut, func(cfg config, w, ew io.Writer) int {
		return runReport(context.Background(), cfg, w, ew, newClient(f))
	})
	var r report
	if e := json.Unmarshal(out.Bytes(), &r); e != nil {
		t.Fatalf("invalid report: %v: %s stderr %s", e, out.String(), errOut.String())
	}
	return code, r, out.String(), errOut.String()
}
func TestJSONMilestoneCommand(t *testing.T) {
	for _, partial := range []bool{false, true} {
		f := &fakeExecutor{replies: []processResult{listFixture([]any{prFixture(1)}, partial, "a")}}
		if partial {
			f.replies = append(f.replies, processResult{Stdout: []byte(`{"errors":[{"message":"failure"}]}`)})
		}
		code, r, out, diag := runFixture(t, []string{"--repo", "org/repo", "--format", "json"}, f)
		want := 0
		if partial {
			want = 3
		}
		if code != want || r.Complete == partial || len(r.PullRequests) != 1 || r.PullRequests[0].UnresolvedThreadsCount == nil {
			t.Fatalf("%d %+v", code, r)
		}
		if !strings.Contains(out, "\n  \"schema_version\": 1") || (diag != "") != partial {
			t.Fatal(out, diag)
		}
		dec := json.NewDecoder(strings.NewReader(out))
		var value any
		if e := dec.Decode(&value); e != nil {
			t.Fatal(e)
		}
		if e := dec.Decode(&value); e != io.EOF {
			t.Fatal("extra output", e)
		}
	}
}
func TestJSONMilestoneUnknownThreads(t *testing.T) {
	p := prFixture(1)
	p["reviewThreads"] = map[string]any{"totalCount": 2}
	f := &fakeExecutor{replies: []processResult{listFixture([]any{p}, false, nil)}}
	code, r, _, _ := runFixture(t, []string{"--repo", "org/repo", "--format", "json"}, f)
	if code != 3 || r.Complete || r.PullRequests[0].UnresolvedThreadsCount != nil || r.PullRequests[0].DetailsComplete {
		t.Fatal(code, r)
	}
}
