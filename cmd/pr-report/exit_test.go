package main

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestReportExitPaths(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var out, diag bytes.Buffer
	if code := run([]string{"--repo", "org/repo"}, &out, &diag, executeReport); code != 1 || out.Len() != 0 || !strings.Contains(diag.String(), "gh") {
		t.Fatal(code, out.String(), diag.String())
	}
	for _, format := range []string{"table", "json"} {
		for _, scenario := range []string{"empty", "blocked", "all fail", "output failure", "cancel"} {
			t.Run(format+scenario, func(t *testing.T) {
				p := prFixture(1)
				p["mergeStateStatus"] = "BLOCKED"
				nodes := []any{p}
				if scenario == "empty" {
					nodes = []any{}
				}
				f := &fakeExecutor{replies: []processResult{listFixture(nodes, false, nil)}}
				if scenario == "all fail" {
					f.replies = []processResult{{Stdout: []byte(`{"data":{"repository":null}}`)}}
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if scenario == "cancel" {
					cancel()
				}
				var out, diag bytes.Buffer
				var w io.Writer = &out
				if scenario == "output failure" {
					w = failingWriter{}
				}
				code := runReport(ctx, config{repos: []string{"org/repo"}, format: format}, w, &diag, newClient(f))
				want := 0
				switch scenario {
				case "all fail":
					want = 3
				case "output failure":
					want = 1
				case "cancel":
					want = 130
				}
				if code != want {
					t.Fatal(code, diag.String())
				}
				if scenario == "all fail" && (!strings.Contains(diag.String(), "not_found_or_forbidden") || out.Len() == 0) {
					t.Fatal(out.String(), diag.String())
				}
				if scenario == "cancel" && (out.Len() != 0 || len(f.calls) != 0) {
					t.Fatal("canceled run produced output or execution")
				}
			})
		}
	}
}
