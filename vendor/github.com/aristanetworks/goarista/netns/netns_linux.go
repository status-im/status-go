// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package netns

import "golang.org/x/sys/unix"

// close closes the file descriptor mapped to a network namespace
func (h nsHandle) close() error {
	return unix.Close(int(h))
}

// fd returns the handle as a uintptr
func (h nsHandle) fd() int {
	return int(h)
}

// getNs returns a file descriptor mapping to the given network namespace
var getNs = func(nsName string) (handle, error) {
	fd, err := unix.Open(nsName, unix.O_RDONLY, 0)
	return nsHandle(fd), err
}

// setNs sets the process's network namespace
var setNs = func(h handle) error {
	return unix.Setns(h.fd(), unix.CLONE_NEWNET)
}
