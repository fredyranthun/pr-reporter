package main

import (
	"fmt"
	"strings"
	"time"
)

// Field names match the public JSON keys. The same catalog drives validation
// and terminal rendering so a selected field means the same thing in both formats.
type fieldSpec struct {
	name, header string
	cell         func(p pullRequest) string
}

var prFields = []fieldSpec{
	{"repo", "REPO", func(p pullRequest) string { return terminalText(p.Repo) }},
	{"number", "PR", func(p pullRequest) string { return fmt.Sprint(p.Number) }},
	{"title", "TÍTULO", func(p pullRequest) string { return tableTitle(p.Title) }},
	{"url", "URL", func(p pullRequest) string { return terminalText(p.URL) }},
	{"author", "AUTOR", func(p pullRequest) string { return terminalText(display(p.Author)) }},
	{"is_draft", "RASCUNHO", func(p pullRequest) string { return displayBlocked(&p.IsDraft) }},
	{"created_at", "CRIADO EM", func(p pullRequest) string { return terminalText(p.CreatedAt) }},
	{"updated_at", "ATUALIZADO EM", func(p pullRequest) string { return terminalText(p.UpdatedAt) }},
	{"observed_at", "OBSERVADO EM", func(p pullRequest) string { return p.ObservedAt.Format(time.RFC3339Nano) }},
	{"conversation_comments_count", "COMENT.", func(p pullRequest) string { return display(p.ConversationCommentsCount) }},
	{"review_threads_count", "THREADS", func(p pullRequest) string { return display(p.ReviewThreadsCount) }},
	{"unresolved_threads_count", "THREADS PEND.", func(p pullRequest) string { return display(p.UnresolvedThreadsCount) }},
	{"has_comments", "TEM COMENT.", func(p pullRequest) string { return displayBlocked(p.HasComments) }},
	{"review_decision", "REVISÃO", func(p pullRequest) string { return terminalText(display(p.ReviewDecision)) }},
	{"mergeable", "MERGEABLE", func(p pullRequest) string { return terminalText(display(p.Mergeable)) }},
	{"merge_state_status", "MERGE", func(p pullRequest) string { return terminalText(display(p.MergeStateStatus)) }},
	{"blocked", "BLOQUEADO", func(p pullRequest) string { return displayBlocked(p.Blocked) }},
	{"signals", "SINAIS", func(p pullRequest) string { return terminalText(strings.Join(p.Signals, ", ")) }},
	{"details_complete", "DETALHES COMPLETOS", func(p pullRequest) string { return displayBlocked(&p.DetailsComplete) }},
}

var defaultTableFields = []string{
	"repo", "number", "title", "conversation_comments_count", "unresolved_threads_count",
	"review_decision", "merge_state_status", "blocked", "url",
}

func fieldNames() string {
	names := make([]string, len(prFields))
	for i, f := range prFields {
		names[i] = f.name
	}
	return strings.Join(names, ", ")
}

func parseFields(value string) ([]string, error) {
	parts := strings.Split(value, ",")
	fields := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			return nil, fmt.Errorf("--fields must contain non-empty comma-separated names; available: %s", fieldNames())
		}
		valid := false
		for _, f := range prFields {
			if f.name == name {
				valid = true
				break
			}
		}
		if !valid {
			return nil, fmt.Errorf("unknown --fields name %q; available: %s", name, fieldNames())
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate --fields name %q", name)
		}
		seen[name] = true
		fields = append(fields, name)
	}
	return fields, nil
}

func fieldByName(name string) fieldSpec {
	for _, f := range prFields {
		if f.name == name {
			return f
		}
	}
	panic("unvalidated field: " + name)
}
