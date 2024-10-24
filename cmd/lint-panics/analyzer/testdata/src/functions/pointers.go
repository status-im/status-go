package functions

import (
	"common"
)

func init() {
	runAsync(ok)
	runAsyncOk(ok)
}

func runAsync(fn func()) {
	go fn() // want "missing defer call to LogOnPanic"
}

func runAsyncOk(fn func()) {
	go func() {
		defer common.LogOnPanic()
		fn()
	}()
}
