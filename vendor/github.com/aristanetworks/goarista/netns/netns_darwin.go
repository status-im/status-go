// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package netns

// stub: close closes the file descriptor mapped to a network namespace
func (h nsHandle) close() error {
	return nil
}

// stub: fd returns the handle as a uintptr
func (h nsHandle) fd() int {
	return 0
}

// stub: getNs returns a file descriptor mapping to the given network namespace
var getNs = func(nsName string) (handle, error) {
	return nsHandle(1), nil
}

// stub: setNs sets the process's network namespace
var setNs = func(fd handle) error {
	return nil
}
