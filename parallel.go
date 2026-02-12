package main

import "sync"

const maxConcurrency = 10

func runParallel[T any](items []T, fn func(T)) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrency)
	for _, item := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(it T) {
			defer wg.Done()
			defer func() { <-sem }()
			fn(it)
		}(item)
	}
	wg.Wait()
}
