package main

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"
)

func TestAPIFieldPresence(t *testing.T) {
	var p apiPR
	if err := json.Unmarshal([]byte(`{"author":null,"reviewDecision":null,"isDraft":false,"comments":{"totalCount":0},"mergeable":"FUTURE"}`), &p); err != nil {
		t.Fatal(err)
	}
	if !p.Author.Present || !p.Author.Null || !p.Review.Null || !p.Draft.known() || p.Draft.Value || !p.Comments.Value.Total.known() || p.Threads.Present || p.Mergeable.Value != "FUTURE" {
		t.Fatalf("presence lost: %+v", p)
	}
	if err := json.Unmarshal([]byte(`{"isDraft":"false"}`), &apiPR{}); err == nil {
		t.Fatal("accepted invalid bool")
	}
}
func jsonObject(t *testing.T, v any) map[string]json.RawMessage {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	var m map[string]json.RawMessage
	if e = json.Unmarshal(b, &m); e != nil {
		t.Fatal(e)
	}
	return m
}
func TestPublicModelJSON(t *testing.T) {
	p := jsonObject(t, pullRequest{})
	want := []string{"repo", "number", "title", "url", "author", "is_draft", "created_at", "updated_at", "observed_at", "conversation_comments_count", "review_threads_count", "unresolved_threads_count", "has_comments", "review_decision", "mergeable", "merge_state_status", "blocked", "signals", "details_complete"}
	got := []string{}
	for k := range p {
		got = append(got, k)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("keys %v", got)
	}
	for _, k := range []string{"author", "conversation_comments_count", "review_threads_count", "unresolved_threads_count", "has_comments", "review_decision", "mergeable", "merge_state_status", "blocked"} {
		if string(p[k]) != "null" {
			t.Fatalf("%s: %s", k, p[k])
		}
	}
	if string(p["signals"]) != "[]" || string(p["is_draft"]) != "false" {
		t.Fatal(p)
	}
	p = jsonObject(t, pullRequest{Blocked: ptr(false), ConversationCommentsCount: ptr(0), Mergeable: ptr("NEW")})
	if string(p["blocked"]) != "false" || string(p["conversation_comments_count"]) != "0" || string(p["mergeable"]) != `"NEW"` {
		t.Fatal(p)
	}
	r := jsonObject(t, newReport(time.Date(2026, 1, 1, 1, 0, 0, 0, time.FixedZone("x", 3600))))
	if len(r) != 11 || string(r["schema_version"]) != "1" || string(r["started_at"]) != `"2026-01-01T00:00:00Z"` {
		t.Fatal(r)
	}
	for _, k := range []string{"repositories", "pull_requests", "errors", "warnings"} {
		if string(r[k]) != "[]" {
			t.Fatal(r)
		}
	}
	d := jsonObject(t, diagnostic{})
	if len(d) != 5 || string(d["repo"]) != "null" || string(d["pr_number"]) != "null" {
		t.Fatal(d)
	}
	rr := jsonObject(t, repositoryResult{RequestedRepo: "ORG/repo"})
	if len(rr) != 5 || string(rr["repo"]) != "null" || string(rr["requested_repo"]) != `"ORG/repo"` {
		t.Fatal(rr)
	}
}
