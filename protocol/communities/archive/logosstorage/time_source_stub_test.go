//go:build use_logos_storage
// +build use_logos_storage

package logosstorage_test

import "time"

type TimeSourceStub struct {
}

func (t *TimeSourceStub) GetCurrentTime() uint64 {
	return uint64(time.Now().Unix())
}
