package main

import (
	"context"
	"maps"
	"time"
)

type client struct {
	exec executor
	now  func() time.Time
}

func newClient(e executor) *client { return &client{exec: e, now: time.Now} }

type collectedPR struct {
	source apiPR
	public pullRequest
}

func publicPR(repo string, p apiPR, observed time.Time) pullRequest {
	out := pullRequest{Repo: repo, Number: p.Number.Value, Title: p.Title.Value, URL: p.URL.Value, IsDraft: p.Draft.Value, CreatedAt: p.Created.Value, UpdatedAt: p.Updated.Value, ObservedAt: observed.UTC(), ConversationCommentsCount: countValue(p.Comments), ReviewThreadsCount: countValue(p.Threads), ReviewDecision: p.Review.pointer(), Mergeable: p.Mergeable.pointer(), MergeStateStatus: p.MergeState.pointer()}
	if p.Author.known() {
		out.Author = p.Author.Value.Login.pointer()
	}
	if out.ReviewThreadsCount != nil && *out.ReviewThreadsCount == 0 {
		out.UnresolvedThreadsCount = ptr(0)
	}
	out.Blocked = classifyMerge(p)
	out.HasComments = commentIndicator(out.ConversationCommentsCount, out.ReviewThreadsCount)
	out.DetailsComplete = len(mainDetails(p)) == 0 && out.UnresolvedThreadsCount != nil
	return out
}
func (c *client) listPRs(ctx context.Context, requested string) (repositoryResult, []collectedPR, []diagnostic) {
	result := repositoryResult{RequestedRepo: requested, DetailsComplete: true}
	var prs []collectedPR
	var errs []diagnostic
	cursor := ""
	cursors := map[string]bool{}
	ids := map[string]int{}
	numbers := map[int]string{}
	fail := func(e *requestFailure) { errs = append(errs, e.diagnostic(ptr(requested), nil, "list_prs")) }
	for ctx.Err() == nil {
		raw := c.exec.Execute(ctx, queryArgs(listQuery, requested, cursor)...)
		observed := c.now().UTC()
		page, e := decodeList(raw)
		if e != nil {
			fail(e)
			break
		}
		info := page.PRs.Value.PageInfo.Value
		if info.HasNext.Value && cursors[info.Cursor.Value] {
			fail(invalid("repeated PR cursor"))
			break
		}
		if result.Repo != nil && *result.Repo != page.Name.Value {
			fail(invalid("canonical repository name changed between pages"))
			break
		}
		nextIDs, nextNumbers := maps.Clone(ids), maps.Clone(numbers)
		for _, node := range page.PRs.Value.Nodes.Value {
			p := node.Value
			id, n := p.ID.Value, p.Number.Value
			if old, ok := nextIDs[id]; ok && old != n {
				e = invalid("PR ID has conflicting numbers")
				break
			}
			if old, ok := nextNumbers[n]; ok && old != id {
				e = invalid("PR number has conflicting IDs")
				break
			}
			nextIDs[id] = n
			nextNumbers[n] = id
		}
		if e != nil {
			fail(e)
			break
		}
		result.Repo = ptr(page.Name.Value)
		for _, node := range page.PRs.Value.Nodes.Value {
			p := node.Value
			if _, ok := ids[p.ID.Value]; ok {
				continue
			}
			ids[p.ID.Value] = p.Number.Value
			numbers[p.Number.Value] = p.ID.Value
			out := publicPR(page.Name.Value, p, observed)
			prs = append(prs, collectedPR{p, out})
			if missing := mainDetails(p); len(missing) > 0 {
				errs = append(errs, detailFailure(missing).diagnostic(result.Repo, ptr(p.Number.Value), "list_prs"))
			}
		}
		if !info.HasNext.Value {
			result.ListingComplete = true
			break
		}
		cursor = info.Cursor.Value
		cursors[cursor] = true
	}
	result.PRsCollected = len(prs)
	for _, p := range prs {
		result.DetailsComplete = result.DetailsComplete && p.public.DetailsComplete
	}
	return result, prs, errs
}
