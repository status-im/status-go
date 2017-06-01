// +build test

package main

import (
	"testing"
)

// the actual test functions are in non-_test.go files (so that they can use cgo i.e. import "C")
// the only intent of these wrappers is for gotest can find what tests are exposed.
func TestExportedAPI(t *testing.T) {
	allTestsDone := make(chan struct{}, 1)
	go testExportedAPI(t, allTestsDone)

	<-allTestsDone
}
