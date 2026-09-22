package main

import "context"

func (c *client) collect(ctx context.Context, cfg config) report {
	r := newReport(c.now())
	r.Filters.OnlyUnresolved = cfg.onlyUnresolved
	r.Complete = true
	for _, requested := range cfg.repos {
		if ctx.Err() != nil {
			r.Complete = false
			break
		}
		repo, prs, errs := c.listPRs(ctx, requested)
		errs = append(errs, c.collectDetails(ctx, prs)...)
		repo.DetailsComplete = true
		for _, p := range prs {
			repo.DetailsComplete = repo.DetailsComplete && p.public.DetailsComplete
		}
		r.Repositories = append(r.Repositories, repo)
		r.Errors = append(r.Errors, errs...)
		r.Complete = r.Complete && repo.ListingComplete && repo.DetailsComplete
		for _, p := range prs {
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
