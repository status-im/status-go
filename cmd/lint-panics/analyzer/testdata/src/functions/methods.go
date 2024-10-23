package functions

import (
	"common"
	"fmt"
)

type Test struct {
}

func init() {
	t := Test{}
	go t.ok()
	go t.empty()
	go t.noLogOnPanic() // want "missing defer call to LogOnPanic"
	go t.notDefer()     // want "missing defer call to LogOnPanic"
}

func (p *Test) ok() {
	defer common.LogOnPanic()
}

func (p *Test) empty() {

}

func (p *Test) noLogOnPanic() {
	defer fmt.Println("FooNoLogOnPanic")
}

func (p *Test) notDefer() {
	common.LogOnPanic()
}
