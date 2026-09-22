package main

import "context"

type repositoryCollection struct {
	result repositoryResult
	prs    []collectedPR
	errors []diagnostic
}

func (c *client) collect(ctx context.Context, cfg config) report {
	r := newReport(c.now())
	r.Filters.OnlyUnresolved = cfg.onlyUnresolved
	concurrency := cfg.concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	c.limit = make(chan struct{}, concurrency)
	results := make([]repositoryCollection, len(cfg.repos))
	listing := func(i int) { v := &results[i]; v.result, v.prs, v.errors = c.listPRs(ctx, cfg.repos[i]) }
	if concurrency == 1 {
		for i := range cfg.repos {
			if ctx.Err() != nil {
				break
			}
			listing(i)
			v := &results[i]
			v.errors = append(v.errors, c.collectDetails(ctx, v.prs)...)
		}
	} else {
		// Two bounded phases avoid nested worker pools. Every connection retains
		// sequential cursors, while detail jobs from a single large repo can overlap.
		parallelFor(ctx, concurrency, len(results), listing)
		type detailJob struct{ repo, pr int }
		jobs := []detailJob{}
		for i, v := range results {
			for j := range v.prs {
				jobs = append(jobs, detailJob{i, j})
			}
		}
		errs := make([][]diagnostic, len(jobs))
		parallelFor(ctx, concurrency, len(jobs), func(i int) { job := jobs[i]; errs[i] = c.collectDetails(ctx, results[job.repo].prs[job.pr:job.pr+1]) })
		for i, job := range jobs {
			results[job.repo].errors = append(results[job.repo].errors, errs[i]...)
		}
	}
	r.Complete = ctx.Err() == nil
	for _, v := range results {
		v.result.DetailsComplete = true
		for _, p := range v.prs {
			v.result.DetailsComplete = v.result.DetailsComplete && p.public.DetailsComplete
		}
		r.Repositories = append(r.Repositories, v.result)
		r.Errors = append(r.Errors, v.errors...)
		r.Complete = r.Complete && v.result.ListingComplete && v.result.DetailsComplete
		for _, p := range v.prs {
			signals, warnings := deriveSignals(p)
			p.public.Signals = signals
			r.Warnings = append(r.Warnings, warnings...)
			r.PullRequests = append(r.PullRequests, p.public)
		}
	}
	sortAndFilter(&r)
	r.FinishedAt = c.now().UTC()
	return r
}
