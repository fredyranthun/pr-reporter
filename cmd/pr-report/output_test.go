package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJSONGoldenReports(t *testing.T) {
	for _, scenario := range []string{"complete", "empty", "filtered", "incomplete", "unknown_enum"} {
		t.Run(scenario, func(t *testing.T) {
			p := prFixture(1)
			p["title"] = "Full title: 日本語 \"quoted\"\n\u001b[31m<&>"
			nodes := []any{p}
			if scenario == "empty" {
				nodes = []any{}
			}
			if scenario == "unknown_enum" {
				p["mergeable"] = "FUTURE_STATE"
			}
			if scenario == "incomplete" {
				delete(p, "comments")
			}
			f := &fakeExecutor{replies: []processResult{listFixture(nodes, false, nil)}}
			c := newClient(f)
			c.now = func() time.Time { return time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC) }
			var out, diag bytes.Buffer
			code := runReport(context.Background(), config{repos: []string{"ORG/repo"}, format: "json", onlyUnresolved: scenario == "filtered"}, &out, &diag, c)
			wantCode := 0
			if scenario == "incomplete" {
				wantCode = 3
			}
			if code != wantCode {
				t.Fatal(code, diag.String())
			}
			path := filepath.Join("testdata", scenario+".json")
			if os.Getenv("PR_REPORT_UPDATE_GOLDEN") == "1" {
				if e := os.MkdirAll("testdata", 0755); e != nil {
					t.Fatal(e)
				}
				if e := os.WriteFile(path, out.Bytes(), 0644); e != nil {
					t.Fatal(e)
				}
			}
			want, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(out.Bytes(), want) {
				t.Fatalf("fixture differs: %s", out.String())
			}
			dec := json.NewDecoder(&out)
			var r report
			if e := dec.Decode(&r); e != nil {
				t.Fatal(e)
			}
			if e := dec.Decode(new(any)); e != io.EOF {
				t.Fatal("trailing output", e)
			}
			if len(r.PullRequests) > 0 && r.PullRequests[0].Title != p["title"] {
				t.Fatal("title changed")
			}
			if scenario == "unknown_enum" && (len(r.Warnings) != 1 || !strings.Contains(diag.String(), "unknown_enum") || *r.PullRequests[0].Mergeable != "FUTURE_STATE") {
				t.Fatal(r, diag.String())
			}
		})
	}
}
