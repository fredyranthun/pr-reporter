package main

import (
	"context"
	"testing"
)

func TestCommentIndicator(t *testing.T) {
	for _, a := range []*int{nil, ptr(0), ptr(2)} {
		for _, b := range []*int{nil, ptr(0), ptr(3)} {
			got := commentIndicator(a, b)
			positive := a != nil && *a > 0 || b != nil && *b > 0
			if positive {
				if got == nil || !*got {
					t.Fatal(a, b, got)
				}
			} else if a != nil && b != nil {
				if got == nil || *got {
					t.Fatal(a, b, got)
				}
			} else if got != nil {
				t.Fatal(a, b, got)
			}
		}
	}
}
func TestReviewBodyOnlyAndResolvedThreads(t *testing.T) {
	for _, threads := range []int{0, 1} {
		p := prFixture(1)
		p["reviews"] = map[string]any{"nodes": []any{map[string]any{"body": "Review text only"}}}
		p["reviewThreads"] = map[string]any{"totalCount": threads}
		f := &fakeExecutor{replies: []processResult{listFixture([]any{p}, false, nil)}}
		if threads > 0 {
			f.replies = append(f.replies, threadFixture([]any{threadNode("resolved", true)}, false, nil))
		}
		r := newClient(f).collect(context.Background(), config{repos: []string{"org/repo"}})
		if !r.Complete || r.PullRequests[0].HasComments == nil || *r.PullRequests[0].HasComments != (threads > 0) || *r.PullRequests[0].UnresolvedThreadsCount != 0 {
			t.Fatal(r)
		}
	}
	// A later unresolved thread cannot fill an unavailable principal count.
	p := prFixture(1)
	delete(p, "reviewThreads")
	f := &fakeExecutor{replies: []processResult{listFixture([]any{p}, false, nil), threadFixture([]any{threadNode("new", false)}, false, nil)}}
	r := newClient(f).collect(context.Background(), config{repos: []string{"org/repo"}})
	if r.PullRequests[0].HasComments != nil || *r.PullRequests[0].UnresolvedThreadsCount != 1 {
		t.Fatal(r)
	}
}

func TestMergeDecisionTable(t *testing.T) {
	cases := []struct {
		name, merge, state string
		draft              bool
		review             any
		want               *bool
	}{
		{"clean", "MERGEABLE", "CLEAN", false, nil, ptr(false)},
		{"review required is signal", "MERGEABLE", "CLEAN", false, "REVIEW_REQUIRED", ptr(false)},
		{"draft", "UNKNOWN", "UNKNOWN", true, nil, ptr(true)},
		{"state draft", "MERGEABLE", "DRAFT", false, nil, ptr(true)},
		{"conflicting", "CONFLICTING", "BEHIND", false, nil, ptr(true)},
		{"dirty", "MERGEABLE", "DIRTY", false, nil, ptr(true)},
		{"blocked without cause", "MERGEABLE", "BLOCKED", false, nil, ptr(true)},
		{"unknown conflict but blocked", "UNKNOWN", "BLOCKED", false, nil, ptr(true)},
		{"unknown clean", "UNKNOWN", "CLEAN", false, nil, nil},
		{"behind", "MERGEABLE", "BEHIND", false, nil, nil},
		{"unstable", "MERGEABLE", "UNSTABLE", false, nil, nil},
		{"hooks", "MERGEABLE", "HAS_HOOKS", false, nil, nil},
		{"unknown", "MERGEABLE", "UNKNOWN", false, nil, nil},
		{"contradictory draft", "MERGEABLE", "CLEAN", true, nil, nil},
		{"contradictory conflicts", "CONFLICTING", "CLEAN", false, nil, nil},
		{"contradictory review", "MERGEABLE", "CLEAN", false, "CHANGES_REQUESTED", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := prFixture(1)
			p["mergeable"] = tc.merge
			p["mergeStateStatus"] = tc.state
			p["isDraft"] = tc.draft
			p["reviewDecision"] = tc.review
			r, e := decodeList(listFixture([]any{p}, false, nil))
			if e != nil {
				t.Fatal(e)
			}
			got := classifyMerge(r.PRs.Value.Nodes.Value[0].Value)
			if (got == nil) != (tc.want == nil) || got != nil && *got != *tc.want {
				t.Fatal(got, tc.want)
			}
		})
	}
}
func TestMergeMissingAndUnknownPrecedence(t *testing.T) {
	for _, key := range []string{"reviewDecision", "mergeable", "mergeStateStatus"} {
		for _, mode := range []string{"absent", "unknown"} {
			for _, state := range []string{"DRAFT", "DIRTY", "BLOCKED", "CLEAN"} {
				p := prFixture(1)
				p["mergeStateStatus"] = state
				if mode == "absent" {
					delete(p, key)
				} else {
					p[key] = "FUTURE"
				}
				r, e := decodeList(listFixture([]any{p}, false, nil))
				if e != nil {
					t.Fatal(e)
				}
				if classifyMerge(r.PRs.Value.Nodes.Value[0].Value) != nil {
					t.Fatal(key, mode, state)
				}
			}
		}
	}
	p := prFixture(1)
	delete(p, "author")
	delete(p, "comments")
	p["mergeStateStatus"] = "BLOCKED"
	r, e := decodeList(listFixture([]any{p}, false, nil))
	if e != nil {
		t.Fatal(e)
	}
	if b := classifyMerge(r.PRs.Value.Nodes.Value[0].Value); b == nil || !*b {
		t.Fatal("unrelated details prevented classification")
	}
	p = prFixture(1)
	p["reviewThreads"] = map[string]any{"totalCount": 1}
	f := &fakeExecutor{replies: []processResult{listFixture([]any{p}, false, nil), threadFixture([]any{threadNode("x", false)}, false, nil)}}
	out := newClient(f).collect(context.Background(), config{repos: []string{"org/repo"}})
	if b := out.PullRequests[0].Blocked; b == nil || *b {
		t.Fatal("unresolved thread established blockage")
	}
}

func TestSignalsAndCompatibilityWarnings(t *testing.T) {
	for _, field := range []string{"mergeable", "mergeStateStatus", "reviewDecision"} {
		for _, state := range []string{"DRAFT", "DIRTY", "BLOCKED", "CLEAN"} {
			p := prFixture(1)
			p["mergeStateStatus"] = state
			p[field] = "NEW_VALUE"
			f := &fakeExecutor{replies: []processResult{listFixture([]any{p}, false, nil)}}
			r := newClient(f).collect(context.Background(), config{repos: []string{"org/repo"}})
			if !r.Complete || !r.PullRequests[0].DetailsComplete || r.PullRequests[0].Blocked != nil || len(r.Warnings) != 1 || r.Warnings[0].Code != "unknown_enum" || *r.Warnings[0].PRNumber != 1 || *r.Warnings[0].Repo != "Org/Repo" {
				t.Fatal(r)
			}
			found := false
			for _, s := range r.PullRequests[0].Signals {
				found = found || s == "merge_unknown"
			}
			if !found {
				t.Fatal(r)
			}
		}
	}
	cases := map[string]string{"BEHIND": "behind", "BLOCKED": "merge_blocked", "DIRTY": "conflicts", "DRAFT": "draft", "HAS_HOOKS": "hooks_present", "UNKNOWN": "merge_unknown", "UNSTABLE": "nonpassing_status"}
	for state, signal := range cases {
		p := prFixture(1)
		p["mergeStateStatus"] = state
		r, _ := decodeList(listFixture([]any{p}, false, nil))
		source := r.PRs.Value.Nodes.Value[0].Value
		s, w := deriveSignals(collectedPR{source: source, public: publicPR("Org/Repo", source, newClient(nil).now())})
		if len(s) != 1 || s[0] != signal || len(w) != 0 {
			t.Fatal(state, s, w)
		}
	}
	p := prFixture(1)
	p["isDraft"] = true
	p["mergeable"] = "CONFLICTING"
	p["reviewDecision"] = "CHANGES_REQUESTED"
	r, _ := decodeList(listFixture([]any{p}, false, nil))
	source := r.PRs.Value.Nodes.Value[0].Value
	out := publicPR("Org/Repo", source, newClient(nil).now())
	out.UnresolvedThreadsCount = ptr(1)
	s, _ := deriveSignals(collectedPR{source, out})
	want := []string{"changes_requested", "conflicts", "draft", "inconsistent_merge_data", "unresolved_threads"}
	if len(s) != len(want) {
		t.Fatal(s)
	}
	for i := range want {
		if s[i] != want[i] {
			t.Fatal(s)
		}
	}
}
