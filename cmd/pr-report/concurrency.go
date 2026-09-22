package main

import (
	"context"
	"sync"
)

func parallelFor(ctx context.Context, limit, count int, fn func(int)) {
	if count < limit {
		limit = count
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if ctx.Err() != nil {
					return
				}
				fn(j)
			}
		}()
	}
dispatch:
	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			break dispatch
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()
}
