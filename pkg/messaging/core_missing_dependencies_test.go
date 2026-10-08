package messaging

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/panics"
)

type fakeMissingDependencyFetcher struct {
	mu        sync.Mutex
	processed map[string]bool
	fetches   [][]string
	// fetchErrs[i] is returned by the i-th fetch; fetches beyond it succeed.
	fetchErrs []error
	// onFetch, if set, runs on every fetch (e.g. to mark hashes as processed).
	onFetch func(attempt int)
}

func (f *fakeMissingDependencyFetcher) AlreadyProcessed(hashes []string) (map[string]bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	hits := make(map[string]bool)
	for _, h := range hashes {
		if f.processed[h] {
			hits[h] = true
		}
	}
	return hits, nil
}

func (f *fakeMissingDependencyFetcher) FetchMessagesByHashes(_ context.Context, hashes []string) error {
	f.mu.Lock()
	attempt := len(f.fetches)
	f.fetches = append(f.fetches, append([]string(nil), hashes...))
	onFetch := f.onFetch
	f.mu.Unlock()
	if onFetch != nil {
		onFetch(attempt)
	}
	if attempt < len(f.fetchErrs) {
		return f.fetchErrs[attempt]
	}
	return nil
}

func (f *fakeMissingDependencyFetcher) markProcessed(h string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.processed == nil {
		f.processed = make(map[string]bool)
	}
	f.processed[h] = true
}

var noRetryDelays = []time.Duration{0, 0, 0}

func TestFetchMissingDependencies_SuccessNoRetry(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}

	fetchMissingDependencies(context.Background(), zap.NewNop(), f, []string{"0x01", "0x02"}, noRetryDelays)

	require.Equal(t, [][]string{{"0x01", "0x02"}}, f.fetches)
}

func TestFetchMissingDependencies_RetriesAfterFailure(t *testing.T) {
	eof := errors.New("EOF")
	f := &fakeMissingDependencyFetcher{fetchErrs: []error{eof, eof}}

	fetchMissingDependencies(context.Background(), zap.NewNop(), f, []string{"0x01"}, noRetryDelays)

	require.Len(t, f.fetches, 3) // two failures, then success
}

func TestFetchMissingDependencies_GivesUpAfterRetries(t *testing.T) {
	eof := errors.New("EOF")
	f := &fakeMissingDependencyFetcher{fetchErrs: []error{eof, eof, eof, eof, eof}}

	fetchMissingDependencies(context.Background(), zap.NewNop(), f, []string{"0x01"}, noRetryDelays)

	require.Len(t, f.fetches, len(noRetryDelays)+1)
}

func TestFetchMissingDependencies_RetrySkipsAlreadyReceived(t *testing.T) {
	f := &fakeMissingDependencyFetcher{fetchErrs: []error{errors.New("EOF")}}
	// The first attempt delivers 0x01 before the connection drops.
	f.onFetch = func(attempt int) {
		if attempt == 0 {
			f.markProcessed("0x01")
		}
	}

	fetchMissingDependencies(context.Background(), zap.NewNop(), f, []string{"0x01", "0x02"}, noRetryDelays)

	require.Equal(t, [][]string{{"0x01", "0x02"}, {"0x02"}}, f.fetches)
}

func TestFetchMissingDependencies_SkipsWhenAllProcessed(t *testing.T) {
	f := &fakeMissingDependencyFetcher{}
	f.markProcessed("0x01")

	fetchMissingDependencies(context.Background(), zap.NewNop(), f, []string{"0x01"}, noRetryDelays)

	require.Empty(t, f.fetches)
}

func TestFetchMissingDependencies_StopsWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := &fakeMissingDependencyFetcher{fetchErrs: []error{errors.New("EOF"), errors.New("EOF")}}
	f.onFetch = func(int) { cancel() }

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer panics.LogOnPanic()
		fetchMissingDependencies(ctx, zap.NewNop(), f, []string{"0x01"}, []time.Duration{time.Hour})
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("fetchMissingDependencies did not stop after context cancellation")
	}
	require.Len(t, f.fetches, 1)
}
