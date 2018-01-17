// +build e2e_test

// Tests in `./lib` package will run only when `e2e_test` build tag is provided.
// It's required to prevent some files from being included in the binary.
// Check out `lib/utils.go` and `lib/librarytest.go` for more details.

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

func TestExportedAPIWithMockedStatusAPI(t *testing.T) {
	testCreateAccountWithMock(t)
	testCreateChildAccountWithMock(t)
	testRecoverAccountWithMock(t)
	testValidateNodeConfigWithMock(t)
}
