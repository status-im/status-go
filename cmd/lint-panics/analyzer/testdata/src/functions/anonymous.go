package functions

import (
	"common"
	"fmt"
)

func init() {
	go func() {
		defer common.LogOnPanic()
	}()

	go func() {

	}()

	go func() { // want "missing defer call to LogOnPanic"
		fmt.Println("anon")
	}()

	go func() { // want "missing defer call to LogOnPanic"
		common.LogOnPanic()
	}()
}
