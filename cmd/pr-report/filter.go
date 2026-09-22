package main

import (
	"sort"
	"strings"
)

func sortAndFilter(r *report) {
	sort.SliceStable(r.PullRequests, func(i, j int) bool {
		a, b := r.PullRequests[i], r.PullRequests[j]
		ar, br := strings.ToLower(a.Repo), strings.ToLower(b.Repo)
		if ar != br {
			return ar < br
		}
		return a.Number < b.Number
	})
	r.PRsCollected = len(r.PullRequests)
	if r.Filters.OnlyUnresolved {
		kept := make(list[pullRequest], 0, len(r.PullRequests))
		for _, p := range r.PullRequests {
			if p.UnresolvedThreadsCount != nil && *p.UnresolvedThreadsCount > 0 {
				kept = append(kept, p)
			}
		}
		r.PullRequests = kept
	}
	r.PRsReturned = len(r.PullRequests)
}
