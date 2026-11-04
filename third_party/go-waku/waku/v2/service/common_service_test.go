package service

import (
	"context"
	"sync"
	"testing"
)

// check if start and stop on common service works in random order
func TestCommonService(t *testing.T) {
	s := NewCommonService()
	wg := &sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		if i%2 == 0 {
			go func() {
				wg.Done()
				_ = s.Start(context.TODO(), func() error { return nil })
			}()
		} else {
			go func() {
				wg.Done()
				go s.Stop(func() {})
			}()
		}
	}
	wg.Wait()
}
