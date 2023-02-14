package server

import (
	"crypto/rand"
	"math/big"
	"testing"
	"time"
)

func TestTimeoutManager(t *testing.T) {
	tm := newTimeoutManager()

	// test 0 timeout means timeout does not occur
	tm.SetTimeout(0)

	// test fuzzing - 0 timeout - multiple sequential calls to random init and stop funcs
	for i := 0; i < 30; i++ {
		b, err := rand.Int(rand.Reader, big.NewInt(2))
		if err != nil {
			t.Error(err)
		}

		if b.Int64() == 1 {
			tm.StartTimeout(t.FailNow)
		} else {
			tm.StopTimeout()
		}
	}

	// test fuzzing - random timeout - multiple sequential calls to random init and stop funcs
	for i := 0; i < 30; i++ {
		b, err := rand.Int(rand.Reader, big.NewInt(2))
		if err != nil {
			t.Error(err)
		}
		to, err := rand.Int(rand.Reader, big.NewInt(11))
		if err != nil {
			t.Error(err)
		}

		tm.SetTimeout(uint(to.Int64() * 10))

		if b.Int64() == 1 {
			tm.StartTimeout(t.FailNow)
		} else {
			tm.StopTimeout()
		}
	}

	// test StopTimeout() prevents termination func
	tm.SetTimeout(20)
	tm.StartTimeout(t.FailNow)
	time.Sleep(10 * time.Millisecond)
	tm.StopTimeout()

	// test StartTimeout() executes termination func on timeout
	ok := false
	tm.SetTimeout(10)
	tm.StartTimeout(func() {
		ok = true
	})
	time.Sleep(20 * time.Millisecond)
	if !ok {
		t.FailNow()
	}

}
