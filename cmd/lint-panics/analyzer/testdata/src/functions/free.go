package functions

import (
	"common"
	"fmt"
)

func init() {
	go ok()
	go empty()
	go noLogOnPanic() // want "missing defer call to LogOnPanic"
	go notDefer()     // want "missing defer call to LogOnPanic"
}

func ok() {
	defer common.LogOnPanic()
}

func empty() {

}

func noLogOnPanic() {
	defer fmt.Println("Bar")
}

func notDefer() {
	common.LogOnPanic()
}
