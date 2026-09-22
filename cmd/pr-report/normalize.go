package main

import (
	"fmt"
	"sort"
)

func commentIndicator(comments, threads *int) *bool {
	if comments != nil && *comments > 0 || threads != nil && *threads > 0 {
		return ptr(true)
	}
	if comments != nil && threads != nil {
		return ptr(false)
	}
	return nil
}

type enumValue struct{ field, value string }

func unknownEnums(p apiPR) []enumValue {
	var unknown []enumValue
	for _, e := range []struct {
		name    string
		value   apiField[string]
		allowed []string
	}{
		{"mergeable", p.Mergeable, []string{"MERGEABLE", "CONFLICTING", "UNKNOWN"}},
		{"mergeStateStatus", p.MergeState, []string{"BEHIND", "BLOCKED", "CLEAN", "DIRTY", "DRAFT", "HAS_HOOKS", "UNKNOWN", "UNSTABLE"}},
		{"reviewDecision", p.Review, []string{"APPROVED", "CHANGES_REQUESTED", "REVIEW_REQUIRED"}},
	} {
		if !e.value.known() {
			continue
		}
		known := false
		for _, v := range e.allowed {
			known = known || e.value.Value == v
		}
		if !known {
			unknown = append(unknown, enumValue{e.name, e.value.Value})
		}
	}
	return unknown
}
func inconsistentMerge(p apiPR) bool {
	return p.MergeState.Value == "CLEAN" && (p.Draft.Value || p.Mergeable.Value == "CONFLICTING" || p.Review.Value == "CHANGES_REQUESTED")
}
func classifyMerge(p apiPR) *bool {
	if !p.Draft.known() || !p.Mergeable.known() || !p.MergeState.known() || !p.Review.Present || len(unknownEnums(p)) > 0 || inconsistentMerge(p) {
		return nil
	}
	if p.Draft.Value || p.MergeState.Value == "DRAFT" || p.Mergeable.Value == "CONFLICTING" || p.MergeState.Value == "DIRTY" || p.MergeState.Value == "BLOCKED" {
		return ptr(true)
	}
	if p.MergeState.Value == "CLEAN" && p.Mergeable.Value == "MERGEABLE" {
		return ptr(false)
	}
	return nil
}

func deriveSignals(p collectedPR) (list[string], []diagnostic) {
	s := p.source
	var signals list[string]
	var warnings []diagnostic
	add := func(ok bool, name string) {
		if ok {
			signals = append(signals, name)
		}
	}
	add(s.Draft.Value || s.MergeState.Value == "DRAFT", "draft")
	add(s.Mergeable.Value == "CONFLICTING" || s.MergeState.Value == "DIRTY", "conflicts")
	add(s.MergeState.Value == "BLOCKED", "merge_blocked")
	add(s.MergeState.Value == "BEHIND", "behind")
	add(s.MergeState.Value == "UNSTABLE", "nonpassing_status")
	add(s.MergeState.Value == "HAS_HOOKS", "hooks_present")
	add(s.Review.Value == "CHANGES_REQUESTED", "changes_requested")
	add(s.Review.Value == "REVIEW_REQUIRED", "review_required")
	add(p.public.UnresolvedThreadsCount != nil && *p.public.UnresolvedThreadsCount > 0, "unresolved_threads")
	unknown := unknownEnums(s)
	add(s.Mergeable.Value == "UNKNOWN" || s.MergeState.Value == "UNKNOWN" || len(unknown) > 0, "merge_unknown")
	add(inconsistentMerge(s), "inconsistent_merge_data")
	for _, e := range unknown {
		warnings = append(warnings, diagnostic{ptr(p.public.Repo), ptr(p.public.Number), "list_prs", "unknown_enum", fmt.Sprintf("unfamiliar %s value %q", e.field, e.value)})
	}
	sort.Strings(signals)
	return signals, warnings
}
