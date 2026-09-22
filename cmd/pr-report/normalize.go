package main

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
