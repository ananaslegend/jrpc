package jrpc

import (
	"context"
	"sync"
)

func workerPoolWithResult[T any](ctx context.Context, workersCount int) (chan<- func() T, <-chan T) {
	jobs := make(chan func() T)
	resultCh := make(chan T)
	wg := &sync.WaitGroup{}
	wg.Add(workersCount)

	for w := 0; w < workersCount; w++ {
		go workerWithResult(ctx, jobs, resultCh, wg)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	return jobs, resultCh
}

func workerWithResult[T any](ctx context.Context, jobs <-chan func() T, result chan<- T, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case j, ok := <-jobs:
			if !ok {
				return
			}

			result <- j()

		case <-ctx.Done():
			return
		}
	}
}
