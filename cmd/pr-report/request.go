package main

import (
	"context"
	"math/rand/v2"
	"time"
)

func (c *client) attempt(ctx context.Context, args ...string) processResult {
	if err := ctx.Err(); err != nil {
		return processResult{Err: err, ExitCode: -1}
	}
	if c.limit != nil {
		select {
		case <-ctx.Done():
			return processResult{Err: ctx.Err(), ExitCode: -1}
		case c.limit <- struct{}{}:
		}
		defer func() { <-c.limit }()
	}
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

func retryJitter() time.Duration { return time.Duration(rand.Int64N(int64(100*time.Millisecond) + 1)) }
func waitContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
func (c *client) request(ctx context.Context, args ...string) processResult {
	for attempt := 0; ; attempt++ {
		r := c.attempt(ctx, args...)
		if ctx.Err() != nil {
			return r
		}
		_, failure := decodeEnvelope(r)
		if failure == nil || !failure.transient || attempt == 2 {
			return r
		}
		if err := c.wait(ctx, time.Duration(attempt+1)*time.Second+c.jitter()); err != nil {
			return processResult{Err: err, ExitCode: -1}
		}
	}
}
