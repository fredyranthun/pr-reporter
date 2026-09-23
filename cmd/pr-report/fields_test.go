package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFieldCatalogMatchesJSONSchema(t *testing.T) {
	full := jsonObject(t, pullRequest{})
	if len(full) != len(prFields) {
		t.Fatalf("catalog has %d fields; JSON has %d", len(prFields), len(full))
	}
	for _, f := range prFields {
		if _, ok := full[f.name]; !ok {
			t.Errorf("catalog field %q missing from JSON", f.name)
		}
	}
}

func TestParseFieldsValidation(t *testing.T) {
	want := []string{"url", "number", "title"}
	cfg, err := parseConfig([]string{"--repo=org/repo", "--fields", " url, number ,title "})
	if err != nil || !reflect.DeepEqual(cfg.fields, want) {
		t.Fatalf("fields = %v, error = %v", cfg.fields, err)
	}
	for _, tc := range []struct {
		value string
		want  string
	}{
		{"", "non-empty"},
		{"repo,", "non-empty"},
		{",repo", "non-empty"},
		{"repo,,title", "non-empty"},
		{"REPO", "unknown"},
		{"pr", "unknown"},
		{"repo,repo", "duplicate"},
	} {
		t.Run(tc.value, func(t *testing.T) {
			var out, diag bytes.Buffer
			calls := 0
			code := run([]string{"--repo=org/repo", "--fields=" + tc.value}, &out, &diag, func(config, io.Writer, io.Writer) int { calls++; return 0 })
			if code != 2 || calls != 0 || out.Len() != 0 || !strings.Contains(diag.String(), tc.want) {
				t.Fatalf("exit = %d, calls = %d, stdout = %q, stderr = %q", code, calls, out.String(), diag.String())
			}
		})
	}
	if _, err := parseConfig([]string{"--repo=org/repo", "--fields=repo", "--fields=title"}); err == nil || !strings.Contains(err.Error(), "only be specified once") {
		t.Fatalf("repeated flag: %v", err)
	}
}

func TestSelectedTableFields(t *testing.T) {
	r := report{PullRequests: list[pullRequest]{{Repo: "Org/Repo", Number: 12, Title: "Fix\n\x1b[31m", Author: nil, Blocked: ptr(false), Signals: list[string]{"draft", "unresolved_threads"}}}, PRsCollected: 1, PRsReturned: 1}
	var out bytes.Buffer
	if err := writeTableFields(&out, r, []string{"blocked", "title", "author", "number", "signals"}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if strings.Fields(lines[0])[0] != "BLOQUEADO" || !strings.Contains(lines[0], "TÍTULO") || !strings.Contains(lines[0], "SINAIS") {
		t.Fatal(lines[0])
	}
	if !strings.Contains(lines[1], "não") || !strings.Contains(lines[1], "Fix") || !strings.Contains(lines[1], "?") || !strings.Contains(lines[1], "12") || !strings.Contains(lines[1], "draft, unresolved_threads") || strings.Contains(out.String(), "\x1b") {
		t.Fatal(out.String())
	}
	if strings.Contains(out.String(), "Org/Repo") || !strings.Contains(out.String(), "1 coletados, 1 exibidos") {
		t.Fatal(out.String())
	}
}

func TestSelectedJSONFieldsPreserveEnvelopeAndOrder(t *testing.T) {
	r := newReport(time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC))
	r.Complete = false
	r.Repositories = list[repositoryResult]{{RequestedRepo: "Org/Repo", ListingComplete: false}}
	r.PRsCollected, r.PRsReturned = 1, 1
	r.PullRequests = list[pullRequest]{{Repo: "Org/Repo", Number: 12, Title: "Full\n日本語", Author: nil, IsDraft: false, Signals: list[string]{}}}
	r.Errors = list[diagnostic]{{Stage: "list_prs", Code: "api", Message: "partial"}}
	var out bytes.Buffer
	if err := writeJSONFields(&out, r, []string{"title", "author", "number", "is_draft", "signals"}); err != nil {
		t.Fatal(err)
	}
	var got struct {
		SchemaVersion int               `json:"schema_version"`
		Complete      bool              `json:"complete"`
		Repositories  []json.RawMessage `json:"repositories"`
		PRsCollected  int               `json:"prs_collected"`
		PRsReturned   int               `json:"prs_returned"`
		PullRequests  []json.RawMessage `json:"pull_requests"`
		Errors        []json.RawMessage `json:"errors"`
		Warnings      []json.RawMessage `json:"warnings"`
	}
	encoded := out.String()
	dec := json.NewDecoder(&out)
	if err := dec.Decode(&got); err != nil {
		t.Fatal(err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		t.Fatalf("trailing JSON: %v", err)
	}
	if got.SchemaVersion != 1 || got.Complete || got.PRsCollected != 1 || got.PRsReturned != 1 || len(got.Repositories) != 1 || len(got.Errors) != 1 || got.Warnings == nil || len(got.PullRequests) != 1 {
		t.Fatalf("lost report metadata: %+v", got)
	}
	wantPR := "{\n      \"title\": \"Full\\n日本語\",\n      \"author\": null,\n      \"number\": 12,\n      \"is_draft\": false,\n      \"signals\": []\n    }"
	if !strings.Contains(string(got.PullRequests[0]), `"author": null`) || !strings.Contains(encoded, wantPR) || strings.Contains(encoded, `"blocked"`) {
		t.Fatalf("projection or order wrong: %s", encoded)
	}
	out.Reset()
	r.PullRequests = nil
	if err := writeJSONFields(&out, r, []string{"repo"}); err != nil || !strings.Contains(out.String(), `"pull_requests": []`) {
		t.Fatalf("empty projection: %v %s", err, out.String())
	}
}

func TestSelectedFieldsEndToEnd(t *testing.T) {
	f := &fakeExecutor{replies: []processResult{listFixture([]any{prFixture(1)}, false, nil)}}
	var out, diag bytes.Buffer
	code := run([]string{"--repo=org/repo", "--format=json", "--fields=number,title"}, &out, &diag, func(cfg config, w, ew io.Writer) int {
		return runReport(context.Background(), cfg, w, ew, newClient(f))
	})
	if code != 0 || diag.Len() != 0 || len(f.calls) != 1 {
		t.Fatalf("exit = %d, calls = %d, stderr = %s", code, len(f.calls), diag.String())
	}
	var body struct {
		PullRequests []map[string]json.RawMessage `json:"pull_requests"`
	}
	if err := json.Unmarshal(out.Bytes(), &body); err != nil || len(body.PullRequests) != 1 || len(body.PullRequests[0]) != 2 || string(body.PullRequests[0]["number"]) != "1" {
		t.Fatalf("selected output = %s, error = %v", out.String(), err)
	}
}
