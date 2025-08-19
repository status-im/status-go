package utils

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestChunkMapFetcher(t *testing.T) {
	tests := []struct {
		name        string
		items       []string
		chunkLimit  int
		fetchFunc   func(context.Context, []string) (map[string]int, error)
		expected    map[string]int
		expectedErr bool
	}{
		{
			name:       "empty items",
			items:      []string{},
			chunkLimit: 2,
			fetchFunc: func(ctx context.Context, chunk []string) (map[string]int, error) {
				result := make(map[string]int)
				for i, item := range chunk {
					result[item] = i
				}
				return result, nil
			},
			expected:    map[string]int{},
			expectedErr: false,
		},
		{
			name:       "single chunk",
			items:      []string{"a", "b"},
			chunkLimit: 5,
			fetchFunc: func(ctx context.Context, chunk []string) (map[string]int, error) {
				result := make(map[string]int)
				for i, item := range chunk {
					result[item] = i
				}
				return result, nil
			},
			expected:    map[string]int{"a": 0, "b": 1},
			expectedErr: false,
		},
		{
			name:       "multiple chunks",
			items:      []string{"a", "b", "c", "d", "e"},
			chunkLimit: 2,
			fetchFunc: func(ctx context.Context, chunk []string) (map[string]int, error) {
				result := make(map[string]int)
				for _, item := range chunk {
					result[item] = len(item)
				}
				return result, nil
			},
			expected:    map[string]int{"a": 1, "b": 1, "c": 1, "d": 1, "e": 1},
			expectedErr: false,
		},
		{
			name:       "error in fetch function",
			items:      []string{"a", "b", "c"},
			chunkLimit: 2,
			fetchFunc: func(ctx context.Context, chunk []string) (map[string]int, error) {
				return nil, errors.New("fetch error")
			},
			expected:    nil,
			expectedErr: true,
		},
		{
			name:       "zero chunk limit returns empty",
			items:      []string{"a", "b"},
			chunkLimit: 0,
			fetchFunc: func(ctx context.Context, chunk []string) (map[string]int, error) {
				result := make(map[string]int)
				for i, item := range chunk {
					result[item] = i
				}
				return result, nil
			},
			expected:    map[string]int{},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ChunkMapFetcher[int](
				context.Background(),
				tt.items,
				tt.chunkLimit,
				0,
				tt.fetchFunc,
			)

			if tt.expectedErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestChunkArrayFetcher(t *testing.T) {
	tests := []struct {
		name        string
		items       []string
		chunkLimit  int
		fetchFunc   func(context.Context, []string) ([]string, error)
		expected    []string
		expectedErr bool
	}{
		{
			name:       "empty items",
			items:      []string{},
			chunkLimit: 2,
			fetchFunc: func(ctx context.Context, chunk []string) ([]string, error) {
				return chunk, nil
			},
			expected:    []string{},
			expectedErr: false,
		},
		{
			name:       "single chunk",
			items:      []string{"a", "b"},
			chunkLimit: 5,
			fetchFunc: func(ctx context.Context, chunk []string) ([]string, error) {
				result := make([]string, len(chunk))
				for i, item := range chunk {
					result[i] = fmt.Sprintf("%s_processed", item)
				}
				return result, nil
			},
			expected:    []string{"a_processed", "b_processed"},
			expectedErr: false,
		},
		{
			name:       "multiple chunks",
			items:      []string{"a", "b", "c", "d", "e"},
			chunkLimit: 2,
			fetchFunc: func(ctx context.Context, chunk []string) ([]string, error) {
				result := make([]string, 0, len(chunk)*2)
				for _, item := range chunk {
					result = append(result, item, item+"_copy")
				}
				return result, nil
			},
			expected:    []string{"a", "a_copy", "b", "b_copy", "c", "c_copy", "d", "d_copy", "e", "e_copy"},
			expectedErr: false,
		},
		{
			name:       "error in fetch function",
			items:      []string{"a", "b", "c"},
			chunkLimit: 2,
			fetchFunc: func(ctx context.Context, chunk []string) ([]string, error) {
				return nil, errors.New("fetch error")
			},
			expected:    nil,
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ChunkArrayFetcher[string](
				context.Background(),
				tt.items,
				tt.chunkLimit,
				0,
				tt.fetchFunc,
			)

			if tt.expectedErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(tt.expected) == 0 && len(result) == 0 {
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
