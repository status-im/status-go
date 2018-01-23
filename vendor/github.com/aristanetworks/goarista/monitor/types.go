// Copyright (C) 2015  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package monitor

import (
	"strconv"
	"sync/atomic"
)

// Uint is a 64-bit unsigned integer variable that satisfies the Var interface.
type Uint struct {
	i uint64
}

func (v *Uint) String() string {
	return strconv.FormatUint(atomic.LoadUint64(&v.i), 10)
}

// Add delta
func (v *Uint) Add(delta uint64) {
	atomic.AddUint64(&v.i, delta)
}

// Get the uint64 stored in the Uint
func (v *Uint) Get() uint64 {
	return atomic.LoadUint64(&v.i)
}

// Set value
func (v *Uint) Set(value uint64) {
	atomic.StoreUint64(&v.i, value)
}
