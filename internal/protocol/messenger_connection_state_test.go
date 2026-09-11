package protocol

import (
	"sync"
	"testing"

	"github.com/status-im/status-go/internal/connection"
)

func TestMessengerConnectionStateAccessorsAreConcurrentSafe(t *testing.T) {
	m := &Messenger{}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)

		go func(i int) {
			defer wg.Done()
			m.setConnectionState(connection.State{Expensive: i%2 == 0})
		}(i)

		go func() {
			defer wg.Done()
			_ = m.isConnectionExpensive()
		}()
	}

	wg.Wait()
}
