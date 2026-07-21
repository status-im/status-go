package sqlite

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenDBConcurrently(t *testing.T) {
	const workers = 32

	start := make(chan struct{})
	errors := make(chan error, workers)
	var waitGroup sync.WaitGroup
	waitGroup.Add(workers)

	for range workers {
		go func() {
			defer waitGroup.Done()
			defer func() {
				if recovered := recover(); recovered != nil {
					errors <- fmt.Errorf("OpenDB panicked: %v", recovered)
				}
			}()

			<-start
			db, err := OpenDB(InMemoryPath, "password", 2)
			if err != nil {
				errors <- err
				return
			}
			errors <- db.Close()
		}()
	}

	close(start)
	waitGroup.Wait()
	close(errors)

	for err := range errors {
		require.NoError(t, err)
	}
}
