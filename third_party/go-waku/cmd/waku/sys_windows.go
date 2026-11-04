//go:build windows

package main

import (
	"math"
)

func getNumFDs() int {
	return math.MaxInt
}
