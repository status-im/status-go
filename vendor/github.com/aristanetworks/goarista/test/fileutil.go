// Copyright (C) 2015  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package test

import (
	"io"
	"os"
	"testing"
)

// CopyFile copies a file
func CopyFile(t *testing.T, srcPath, dstPath string) {
	src, err := os.Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dst, err := os.Create(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	if err != nil {
		t.Fatal(err)
	}
}
