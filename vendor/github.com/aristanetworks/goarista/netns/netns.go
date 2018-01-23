// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

// Package netns provides a utility function that allows a user to
// perform actions in a different network namespace
package netns

import (
	"fmt"
	"os"
	"runtime"
)

const (
	netNsRunDir = "/var/run/netns/"
	selfNsFile  = "/proc/self/ns/net"
)

// Callback is a function that gets called in a given network namespace.
// The user needs to check any errors from any calls inside this function.
type Callback func() error

// File descriptor interface so we can mock for testing
type handle interface {
	close() error
	fd() int
}

// The file descriptor associated with a network namespace
type nsHandle int

// setNsByName wraps setNs, allowing specification of the network namespace by name.
// It returns the file descriptor mapped to the given network namespace.
func setNsByName(nsName string) error {
	netPath := netNsRunDir + nsName
	handle, err := getNs(netPath)
	if err != nil {
		return fmt.Errorf("Failed to getNs: %s", err)
	}
	err = setNs(handle)
	handle.close()
	if err != nil {
		return fmt.Errorf("Failed to setNs: %s", err)
	}
	return nil
}

// Do takes a function which it will call in the network namespace specified by nsName.
// The goroutine that calls this will lock itself to its current OS thread, hop
// namespaces, call the given function, hop back to its original namespace, and then
// unlock itself from its current OS thread.
// Do returns an error if an error occurs at any point besides in the invocation of
// the given function, or if the given function itself returns an error.
//
// The callback function is expected to do something simple such as just
// creating a socket / opening a connection, as it's not desirable to start
// complex logic in a goroutine that is pinned to the current OS thread.
// Also any goroutine started from the callback function may or may not
// execute in the desired namespace.
func Do(nsName string, cb Callback) error {
	// If destNS is empty, the function is called in the caller's namespace
	if nsName == "" {
		cb()
		return nil
	}

	// Get the file descriptor to the current namespace
	currNsFd, err := getNs(selfNsFile)
	if os.IsNotExist(err) {
		return fmt.Errorf("File descriptor to current namespace does not exist: %s", err)
	} else if err != nil {
		return fmt.Errorf("Failed to open %s: %s", selfNsFile, err)
	}
	defer currNsFd.close()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Jump to the new network namespace
	if err := setNsByName(nsName); err != nil {
		return fmt.Errorf("Failed to set the namespace to %s: %s", nsName, err)
	}

	// Call the given function
	cbErr := cb()

	// Come back to the original namespace
	if err = setNs(currNsFd); err != nil {
		return fmt.Errorf("Failed to return to the original namespace: %s (callback returned %v)",
			err, cbErr)
	}

	return cbErr
}
