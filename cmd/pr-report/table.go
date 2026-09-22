package main

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

func display[T any](p *T) string {
	if p == nil {
		return "?"
	}
	return fmt.Sprint(*p)
}
func displayBlocked(b *bool) string {
	if b == nil {
		return "?"
	}
	if *b {
		return "sim"
	}
	return "não"
}
func writeTable(w io.Writer, r report) error {
	var b strings.Builder
	b.WriteString("REPO\tPR\tTÍTULO\tCOMENT.\tTHREADS PEND.\tREVISÃO\tMERGE\tBLOQUEADO\tURL\n")
	for _, p := range r.PullRequests {
		fmt.Fprintf(&b, "%s\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", terminalText(p.Repo), p.Number, tableTitle(p.Title), display(p.ConversationCommentsCount), display(p.UnresolvedThreadsCount), terminalText(display(p.ReviewDecision)), terminalText(display(p.MergeStateStatus)), displayBlocked(p.Blocked), terminalText(p.URL))
	}
	if len(r.PullRequests) == 0 {
		switch {
		case r.PRsCollected == 0 && !r.Complete:
			b.WriteString("Nenhum PR recuperado; consulta incompleta.\n")
		case r.PRsCollected == 0:
			b.WriteString("Nenhum PR aberto.\n")
		default:
			b.WriteString("Nenhum PR atende ao filtro.\n")
		}
	}
	complete := 0
	for _, repo := range r.Repositories {
		if repo.ListingComplete && repo.DetailsComplete {
			complete++
		}
	}
	fmt.Fprintf(&b, "Repositórios: %d completos, %d incompletos. PRs: %d coletados, %d exibidos.\n", complete, len(r.Repositories)-complete, r.PRsCollected, r.PRsReturned)
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	if _, e := io.WriteString(tw, b.String()); e != nil {
		return e
	}
	return tw.Flush()
}
