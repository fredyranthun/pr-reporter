package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestChatDefaultOutput(t *testing.T) {
	r := report{
		Complete:     true,
		Repositories: list[repositoryResult]{{ListingComplete: true, DetailsComplete: true}, {ListingComplete: true, DetailsComplete: true}},
		PRsCollected: 2,
		PRsReturned:  2,
		PullRequests: list[pullRequest]{
			{
				Repo: "Org/Backend", Number: 12, Title: "Fix login", URL: "https://github.com/Org/Backend/pull/12",
				Author: ptr("alice"), IsDraft: true, ConversationCommentsCount: ptr(2),
				UnresolvedThreadsCount: ptr(1), ReviewDecision: ptr("CHANGES_REQUESTED"),
				MergeStateStatus: ptr("BLOCKED"), Blocked: ptr(true),
			},
			{
				Repo: "Org/Frontend", Number: 4, Title: "Update UI", URL: "https://github.com/Org/Frontend/pull/4",
				Author: ptr("bob"), ConversationCommentsCount: ptr(0),
				UnresolvedThreadsCount: ptr(0), ReviewDecision: ptr("APPROVED"),
				MergeStateStatus: ptr("CLEAN"), Blocked: ptr(false),
			},
		},
	}
	var out bytes.Buffer
	if err := writeChatFields(&out, r, nil); err != nil {
		t.Fatal(err)
	}
	want := `📋 PRs abertos · 2 exibidos de 2 coletados
✅ Coleta completa · 2 repositórios

📁 Org/Backend
• #12 Fix login
  👤 @alice · 📝 Rascunho · ⛔ Bloqueado
  💬 2 comentários · 🧵 1 thread pendente
  Revisão: CHANGES_REQUESTED · Merge: BLOCKED
  https://github.com/Org/Backend/pull/12

📁 Org/Frontend
• #4 Update UI
  👤 @bob · Bloqueado: não
  💬 0 comentários · 🧵 0 threads pendentes
  Revisão: APPROVED · Merge: CLEAN
  https://github.com/Org/Frontend/pull/4
`
	if out.String() != want {
		t.Fatalf("chat output mismatch:\n%s", out.String())
	}
}

func TestChatEmptyAndPartialReports(t *testing.T) {
	for _, tc := range []struct {
		name      string
		complete  bool
		collected int
		message   string
	}{
		{"empty", true, 0, "Nenhum PR aberto."},
		{"partial empty", false, 0, "Nenhum PR recuperado; consulta incompleta."},
		{"filtered", true, 2, "Nenhum PR atende ao filtro."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := report{
				Complete: tc.complete, PRsCollected: tc.collected,
				Repositories: list[repositoryResult]{{ListingComplete: tc.complete, DetailsComplete: true}},
				Filters:      reportFilters{OnlyUnresolved: true},
			}
			var out bytes.Buffer
			if err := writeChatFields(&out, r, nil); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), tc.message) || !strings.Contains(out.String(), "🔎 Filtro: threads pendentes") {
				t.Fatal(out.String())
			}
			if !tc.complete && !strings.Contains(out.String(), "⚠️ Coleta incompleta · 1/1 repositórios incompletos") {
				t.Fatal(out.String())
			}
		})
	}
}

func TestChatFieldsOrderFullTitleAndSanitization(t *testing.T) {
	title := strings.Repeat("界", 61) + "\n@all\x1b[31m"
	r := report{
		Complete: true, PRsCollected: 1, PRsReturned: 1,
		PullRequests: list[pullRequest]{{Repo: "SECRET", Number: 12, Title: title, URL: "https://example.test/pr/12", Signals: list[string]{}}},
	}
	var out bytes.Buffer
	if err := writeChatFields(&out, r, []string{"url", "title", "signals", "blocked"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "SECRET") || strings.Contains(got, "#12") || strings.Contains(got, "\x1b") || strings.Contains(got, "\n@all") {
		t.Fatal(got)
	}
	if !strings.Contains(got, strings.Repeat("界", 61)+"@all") || strings.Contains(got, "…") {
		t.Fatal("title was truncated or not sanitized:", got)
	}
	if !(strings.Index(got, "url:") < strings.Index(got, "title:") &&
		strings.Index(got, "title:") < strings.Index(got, "signals:") &&
		strings.Index(got, "signals:") < strings.Index(got, "blocked:")) {
		t.Fatal("field order lost:", got)
	}
	if !strings.Contains(got, "signals: []") || !strings.Contains(got, "blocked: ?") {
		t.Fatal(got)
	}
}

func TestChatEndToEndAndWriteFailure(t *testing.T) {
	cfg, err := parseConfig([]string{"--repo=org/repo", "--format=chat", "--fields=title,url"})
	if err != nil || cfg.format != "chat" {
		t.Fatal(cfg, err)
	}
	f := &fakeExecutor{replies: []processResult{listFixture([]any{prFixture(1)}, false, nil)}}
	var out, diag bytes.Buffer
	code := runReport(context.Background(), cfg, &out, &diag, newClient(f))
	if code != 0 || diag.Len() != 0 || len(f.calls) != 1 || !strings.Contains(out.String(), "📋 PRs abertos") ||
		!strings.Contains(out.String(), "• title: ") || !strings.Contains(out.String(), "  url: ") {
		t.Fatal(code, out.String(), diag.String())
	}
	if code := runReport(context.Background(), cfg, failingWriter{}, &diag, newClient(&fakeExecutor{replies: []processResult{listFixture([]any{}, false, nil)}})); code != 1 ||
		!strings.Contains(diag.String(), "write output") {
		t.Fatal(code, diag.String())
	}
}
