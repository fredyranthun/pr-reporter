package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestSequentialCommandEndToEnd(t *testing.T) {
	for _, format := range []string{"table", "json"} {
		for _, partial := range []bool{false, true} {
			for _, filtered := range []bool{false, true} {
				p := prFixture(2)
				p["reviewThreads"] = map[string]any{"totalCount": 1}
				p["title"] = "Needs\nreview\x1b[31m"
				f := &fakeExecutor{replies: []processResult{listFixture([]any{p, prFixture(1)}, false, nil), threadFixture([]any{threadNode("pending", false)}, false, nil)}}
				second := listFixture([]any{}, false, nil)
				second.Stdout = []byte(strings.ReplaceAll(string(second.Stdout), "Org/Repo", "Other/Repo"))
				if partial {
					second = processResult{Stdout: []byte(`{"errors":[{"type":"NOT_FOUND"}]}`)}
				}
				f.replies = append(f.replies, second)
				file := writeRepositoryFile(t, "\ufeff# repos\r\nORG/repo\r\norg/REPO\r\n")
				args := []string{"--repos", file, "--repo", "other/repo"}
				if format == "json" {
					args = append(args, "--format", "json")
				}
				if filtered {
					args = append(args, "--only-unresolved")
				}
				var out, diag bytes.Buffer
				code := run(args, &out, &diag, func(cfg config, w, ew io.Writer) int {
					return runReport(context.Background(), cfg, w, ew, newClient(f))
				})
				want := 0
				if partial {
					want = 3
				}
				if code != want || len(f.calls) != 3 {
					t.Fatal(code, len(f.calls), diag.String())
				}
				if format == "json" {
					var r report
					if e := json.Unmarshal(out.Bytes(), &r); e != nil {
						t.Fatal(e)
					}
					n := 2
					if filtered {
						n = 1
					}
					if r.PRsCollected != 2 || r.PRsReturned != n || r.Complete == partial || len(r.Repositories) != 2 || r.PullRequests[len(r.PullRequests)-1].Signals[0] != "unresolved_threads" {
						t.Fatal(r)
					}
				} else {
					if !strings.Contains(out.String(), "THREADS PEND.") || !strings.Contains(out.String(), "Needsreview") || strings.Contains(out.String(), "\x1b") {
						t.Fatal(out.String())
					}
					if partial && !strings.Contains(out.String(), "1 incompletos") {
						t.Fatal(out.String())
					}
				}
				if (diag.Len() > 0) != partial {
					t.Fatal(diag.String())
				}
			}
		}
	}
}
func TestInformationalAndInvalidCommandBypassCollection(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"--version"}, {"--repo", "https://github.com/org/repo"}, {"--repo", "org/repo", "--repo", "invalid"}} {
		f := &fakeExecutor{}
		var out, diag bytes.Buffer
		code := run(args, &out, &diag, func(cfg config, w, ew io.Writer) int {
			return runReport(context.Background(), cfg, w, ew, newClient(f))
		})
		if len(f.calls) != 0 {
			t.Fatal(f.calls)
		}
		if args[0] == "--repo" {
			if code != 2 || out.Len() != 0 {
				t.Fatal(code)
			}
		} else if code != 0 {
			t.Fatal(code)
		}
	}
}
