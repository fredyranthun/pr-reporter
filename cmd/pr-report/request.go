package main

import "context"

func (c *client) attempt(ctx context.Context, args ...string) processResult {
	if err := ctx.Err(); err != nil {
		return processResult{Err: err, ExitCode: -1}
	}
	attempt, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	r := c.exec.Execute(attempt, args...)
	if err := attempt.Err(); err != nil {
		r.Err = err
		r.ExitCode = -1
	}
	return r
}
