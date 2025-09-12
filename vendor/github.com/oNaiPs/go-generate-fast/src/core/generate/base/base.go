// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package base defines shared basic pieces of the go command,
// in particular logging and the Command structure.
package base

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	// "cmd/go/internal/cfg"
	// "cmd/go/internal/str"
)

var atExitFuncs []func()

func AtExit(f func()) {
	atExitFuncs = append(atExitFuncs, f)
}

func Exit() {
	for _, f := range atExitFuncs {
		f()
	}
	os.Exit(exitStatus)
}

func Fatalf(format string, args ...any) {
	Errorf(format, args...)
	Exit()
}

func Errorf(format string, args ...any) {
	log.Printf(format, args...)
	SetExitStatus(1)
}

func ExitIfErrors() {
	if exitStatus != 0 {
		Exit()
	}
}

func Error(err error) {
	// We use errors.Join to return multiple errors from various routines.
	// If we receive multiple errors joined with a basic errors.Join,
	// handle each one separately so that they all have the leading "go: " prefix.
	// A plain interface check is not good enough because there might be
	// other kinds of structured errors that are logically one unit and that
	// add other context: only handling the wrapped errors would lose
	// that context.
	if err != nil && reflect.TypeOf(err).String() == "*errors.joinError" {
		for _, e := range err.(interface{ Unwrap() []error }).Unwrap() {
			Error(e)
		}
		return
	}
	Errorf("go: %v", err)
}

func Fatal(err error) {
	Error(err)
	Exit()
}

var exitStatus = 0
var exitMu sync.Mutex

func SetExitStatus(n int) {
	exitMu.Lock()
	if exitStatus < n {
		exitStatus = n
	}
	exitMu.Unlock()
}

func GetExitStatus() int {
	return exitStatus
}

// AppendPWD returns the result of appending PWD=dir to the environment base.
//
// The resulting environment makes os.Getwd more efficient for a subprocess
// running in dir, and also improves the accuracy of paths relative to dir
// if one or more elements of dir is a symlink.
func AppendPWD(base []string, dir string) []string {
	// POSIX requires PWD to be absolute.
	// Internally we only use absolute paths, so dir should already be absolute.
	if !filepath.IsAbs(dir) {
		panic(fmt.Sprintf("AppendPWD with relative path %q", dir))
	}
	return append(base, "PWD="+dir)
}

// AppendPATH returns the result of appending PATH=$GOROOT/bin:$PATH
// (or the platform equivalent) to the environment base.
func AppendPATH(base []string) []string {
	GOROOTbin := filepath.Join(runtime.GOROOT(), "bin")

	pathVar := "PATH"
	if runtime.GOOS == "plan9" {
		pathVar = "path"
	}

	path := os.Getenv(pathVar)
	if path == "" {
		return append(base, pathVar+"="+GOROOTbin)
	}
	return append(base, pathVar+"="+GOROOTbin+string(os.PathListSeparator)+path)
}
