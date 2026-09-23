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
	return writeTableFields(w, r, nil)
}

func writeTableFields(w io.Writer, r report, names []string) error {
	if names == nil {
		names = defaultTableFields
	}
	var b strings.Builder
	for i, name := range names {
		if i > 0 {
			b.WriteByte('\t')
		}
		b.WriteString(fieldByName(name).header)
	}
	b.WriteByte('\n')
	for _, p := range r.PullRequests {
		for i, name := range names {
			if i > 0 {
				b.WriteByte('\t')
			}
			b.WriteString(fieldByName(name).cell(p))
		}
		b.WriteByte('\n')
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
