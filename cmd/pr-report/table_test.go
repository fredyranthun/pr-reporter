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

func TestTerminalSanitization(t *testing.T) {
	cases := map[string]string{
		"a\t\nb\r\x00c":      "abc",
		"\x1b[31mred\x1b[0m": "red",
		"\x1b]8;;https://example\x1b\\link\x1b]8;;\x1b\\": "link",
		"a\x1b]0;title\ab":      "ab",
		"a\x1bPpayload\x1b\\b":  "ab",
		"a\u009b31mb\u009b0m":   "ab",
		"a\u009dpayload\u009cb": "ab",
		"a\u202eb\u2028c":       "abc",
		"safe\x1b[31":           "safe",
		"safe\x1b":              "safe",
	}
	for input, want := range cases {
		if got := terminalText(input); got != want {
			t.Errorf("%q got %q want %q", input, got, want)
		}
	}
	title := strings.Repeat("界", 61)
	got := tableTitle(title)
	if len([]rune(got)) != 60 || !strings.HasSuffix(got, "…") {
		t.Fatal(got)
	}
	if tableTitle(strings.Repeat("界", 60)) != strings.Repeat("界", 60) {
		t.Fatal("truncated boundary")
	}
	raw := "\x1b[31mVALUE\x1b[0m\t\n"
	r := report{PullRequests: list[pullRequest]{{Repo: raw, Title: raw, ReviewDecision: ptr(raw), MergeStateStatus: ptr(raw), URL: raw}}}
	var out bytes.Buffer
	if e := writeTable(&out, r); e != nil {
		t.Fatal(e)
	}
	if strings.ContainsAny(out.String(), "\x1b\t") || strings.Count(out.String(), "VALUE") != 5 || strings.Count(out.String(), "\n") != 2 {
		t.Fatal(out.String())
	}
	out.Reset()
	if e := writeJSON(&out, r); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(out.String(), `\u001b[31mVALUE\u001b[0m\t\n`) {
		t.Fatal("JSON altered", out.String())
	}
}
