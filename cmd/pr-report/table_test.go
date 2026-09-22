package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestTableColumnsAndNulls(t *testing.T) {
	r := report{PullRequests: list[pullRequest]{{Repo: "Org/Repo", Number: 12, Title: "Fix", ConversationCommentsCount: ptr(2), ReviewThreadsCount: ptr(99), UnresolvedThreadsCount: ptr(3), ReviewDecision: ptr("APPROVED"), MergeStateStatus: ptr("BLOCKED"), Blocked: ptr(true), URL: "https://example/pr/12"}, {Repo: "Org/Repo", Number: 13, Title: "Unknown", Blocked: nil}, {Repo: "Org/Repo", Number: 14, Title: "Ready", Blocked: ptr(false)}}}
	var b bytes.Buffer
	if e := writeTable(&b, r); e != nil {
		t.Fatal(e)
	}
	lines := strings.Split(strings.TrimSpace(b.String()), "\n")
	for _, header := range []string{"REPO", "PR", "TÍTULO", "COMENT.", "THREADS PEND.", "REVISÃO", "MERGE", "BLOQUEADO", "URL"} {
		if !strings.Contains(lines[0], header) {
			t.Fatal(lines[0])
		}
	}
	want := []string{"Org/Repo", "12", "Fix", "2", "3", "APPROVED", "BLOCKED", "sim", "https://example/pr/12"}
	fields := strings.Fields(lines[1])
	for i := range want {
		if fields[i] != want[i] {
			t.Fatal(fields)
		}
	}
	if strings.Count(lines[2], "?") != 5 || !strings.Contains(lines[3], "não") || strings.Contains(b.String(), "99") || strings.Contains(b.String(), "\x1b") {
		t.Fatal(b.String())
	}
}
