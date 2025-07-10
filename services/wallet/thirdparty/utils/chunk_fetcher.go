package utils

import (
	"context"
	"fmt"
	"time"
)

func processInChunks[T any](
	ctx context.Context,
	items []string,
	chunkLimit int,
	delay time.Duration,
	fetchFunc func(context.Context, []string) (T, error),
) ([]T, error) {
	if len(items) == 0 || chunkLimit <= 0 {
		return make([]T, 0), nil
	}

	var results []T
	isFirst := true

	for start := 0; start < len(items); start += chunkLimit {
		end := start + chunkLimit
		if end > len(items) {
			end = len(items)
		}
		chunk := items[start:end]

		if delay > 0 && !isFirst {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
		isFirst = false

		chunkResult, err := fetchFunc(ctx, chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch chunk: %w", err)
		}

		results = append(results, chunkResult)
	}

	return results, nil
}

func ChunkMapFetcher[T any](
	ctx context.Context,
	items []string,
	chunkLimit int,
	delay time.Duration,
	fetchFunc func(context.Context, []string) (map[string]T, error),
) (map[string]T, error) {
	chunkResults, err := processInChunks(ctx, items, chunkLimit, delay, fetchFunc)
	if err != nil {
		return nil, err
	}

	result := make(map[string]T)
	for _, chunkResult := range chunkResults {
		for k, v := range chunkResult {
			result[k] = v
		}
	}

	return result, nil
}

func ChunkArrayFetcher[T any](
	ctx context.Context,
	items []string,
	chunkLimit int,
	delay time.Duration,
	fetchFunc func(context.Context, []string) ([]T, error),
) ([]T, error) {
	chunkResults, err := processInChunks(ctx, items, chunkLimit, delay, fetchFunc)
	if err != nil {
		return nil, err
	}

	var result []T
	for _, chunkResult := range chunkResults {
		result = append(result, chunkResult...)
	}

	return result, nil
}
