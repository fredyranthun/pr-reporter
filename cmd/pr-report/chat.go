package main

import (
	"fmt"
	"io"
	"strings"
)

// Chat uses plain text, Unicode markers, and bare URLs so the same pasted
// report stays readable in Slack and Teams regardless of composer settings.
func writeChatFields(w io.Writer, r report, names []string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "📋 PRs abertos · %d exibidos de %d coletados\n", r.PRsReturned, r.PRsCollected)
	completeRepos := 0
	for _, repo := range r.Repositories {
		if repo.ListingComplete && repo.DetailsComplete {
			completeRepos++
		}
	}
	if r.Complete {
		fmt.Fprintf(&b, "✅ Coleta completa · %d repositórios\n", len(r.Repositories))
	} else {
		fmt.Fprintf(&b, "⚠️ Coleta incompleta · %d/%d repositórios incompletos · resultados parciais\n",
			len(r.Repositories)-completeRepos, len(r.Repositories))
	}
	if r.Filters.OnlyUnresolved {
		b.WriteString("🔎 Filtro: threads pendentes\n")
	}
	if len(r.PullRequests) == 0 {
		b.WriteByte('\n')
		switch {
		case r.PRsCollected == 0 && !r.Complete:
			b.WriteString("Nenhum PR recuperado; consulta incompleta.\n")
		case r.PRsCollected == 0:
			b.WriteString("Nenhum PR aberto.\n")
		default:
			b.WriteString("Nenhum PR atende ao filtro.\n")
		}
		_, err := io.WriteString(w, b.String())
		return err
	}

	lastRepo := ""
	for i, p := range r.PullRequests {
		b.WriteByte('\n')
		if names != nil {
			for j, name := range names {
				prefix := "  "
				if j == 0 {
					prefix = "• "
				}
				fmt.Fprintf(&b, "%s%s: %s\n", prefix, name, chatFieldValue(p, name))
			}
			continue
		}
		repo := terminalText(p.Repo)
		if i == 0 || repo != lastRepo {
			fmt.Fprintf(&b, "📁 %s\n", repo)
			lastRepo = repo
		}
		fmt.Fprintf(&b, "• #%d %s\n", p.Number, terminalText(p.Title))
		author := "?"
		if p.Author != nil {
			author = "@" + terminalText(*p.Author)
		}
		status := "Bloqueado: " + displayBlocked(p.Blocked)
		if p.Blocked != nil && *p.Blocked {
			status = "⛔ Bloqueado"
		}
		if p.IsDraft {
			status = "📝 Rascunho · " + status
		}
		fmt.Fprintf(&b, "  👤 %s · %s\n", author, status)
		fmt.Fprintf(&b, "  💬 %s · 🧵 %s\n",
			chatCount(p.ConversationCommentsCount, "comentário", "comentários"),
			chatCount(p.UnresolvedThreadsCount, "thread pendente", "threads pendentes"))
		fmt.Fprintf(&b, "  Revisão: %s · Merge: %s\n",
			terminalText(display(p.ReviewDecision)), terminalText(display(p.MergeStateStatus)))
		fmt.Fprintf(&b, "  %s\n", terminalText(p.URL))
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func chatCount(n *int, singular, plural string) string {
	if n == nil {
		return "? " + plural
	}
	if *n == 1 {
		return "1 " + singular
	}
	return fmt.Sprintf("%d %s", *n, plural)
}

func chatFieldValue(p pullRequest, name string) string {
	switch name {
	case "title":
		return terminalText(p.Title)
	case "signals":
		if len(p.Signals) == 0 {
			return "[]"
		}
	}
	return fieldByName(name).cell(p)
}
