package main

import (
	"context"
	"strconv"
)

func decodeThreads(raw processResult) (apiConnection[apiThread], *requestFailure) {
	r, e := decodeEnvelope(raw)
	if e != nil {
		return apiConnection[apiThread]{}, e
	}
	if !r.PR.known() {
		return apiConnection[apiThread]{}, invalid("pull request unavailable during thread enumeration")
	}
	f := r.PR.Value.Threads
	if e = validateConnection(f); e != nil {
		return f.Value, e
	}
	for _, n := range f.Value.Nodes.Value {
		if !n.Value.ID.known() || n.Value.ID.Value == "" || !n.Value.Resolved.known() {
			return f.Value, invalid("invalid thread ID or isResolved")
		}
	}
	return f.Value, nil
}
func (c *client) countThreads(ctx context.Context, p pullRequest) (*int, *requestFailure) {
	if p.ReviewThreadsCount != nil && *p.ReviewThreadsCount == 0 {
		return ptr(0), nil
	}
	cursor := ""
	cursors := map[string]bool{}
	seen := map[string]bool{}
	count := 0
	for ctx.Err() == nil {
		args := append(queryArgs(threadQuery, p.Repo, cursor), "-F", "number="+strconv.Itoa(p.Number))
		page, e := decodeThreads(c.attempt(ctx, args...))
		if e != nil {
			return nil, e
		}
		info := page.PageInfo.Value
		if info.HasNext.Value && cursors[info.Cursor.Value] {
			return nil, invalid("repeated thread cursor")
		}
		for _, n := range page.Nodes.Value {
			id := n.Value.ID.Value
			if seen[id] {
				continue
			}
			seen[id] = true
			if !n.Value.Resolved.Value {
				count++
			}
		}
		if !info.HasNext.Value {
			return ptr(count), nil
		}
		cursor = info.Cursor.Value
		cursors[cursor] = true
	}
	return nil, &requestFailure{"api", "thread collection canceled", false}
}
func (c *client) collectDetails(ctx context.Context, prs []collectedPR) []diagnostic {
	var errs []diagnostic
	for i := range prs {
		if ctx.Err() != nil {
			break
		}
		p := &prs[i]
		count, e := c.countThreads(ctx, p.public)
		p.public.UnresolvedThreadsCount = count
		p.public.DetailsComplete = len(mainDetails(p.source)) == 0 && e == nil
		if e != nil {
			errs = append(errs, e.diagnostic(ptr(p.public.Repo), ptr(p.public.Number), "review_threads"))
		}
	}
	return errs
}
