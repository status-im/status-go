//go:build !linux && !darwin && !windows

package main

import "runtime"

// TODO: figure out how to get the number of file descriptors on Windows and other systems
func getNumFDs() int {
	fmt.Println("cannot determine number of file descriptors on ", runtime.GOOS)
	return 0
}
