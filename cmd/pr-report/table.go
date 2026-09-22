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
		fmt.Fprintf(&b, "%s\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", p.Repo, p.Number, p.Title, display(p.ConversationCommentsCount), display(p.UnresolvedThreadsCount), display(p.ReviewDecision), display(p.MergeStateStatus), displayBlocked(p.Blocked), p.URL)
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	if _, e := io.WriteString(tw, b.String()); e != nil {
		return e
	}
	return tw.Flush()
}
