//go:build linux || darwin

package main

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func getNumFDs() int {
	var l unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &l); err != nil {
		fmt.Println("failed to get fd limit:" + err.Error())
		return 0
	}
	return int(l.Cur)
}
