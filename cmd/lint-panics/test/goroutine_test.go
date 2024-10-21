package test

import (
	gocommon "github.com/status-im/status-go/common"
	"fmt"
)

type Test struct {
}

func init() {
	t := Test{}
	go t.Empty()
	go t.Foo()
	go t.FooOK()
	go t.FooNotDefer()

	go Empty()
	go Bar()
	go BarOK()
	go BarNotDefer()

	go func() {
		defer gocommon.LogOnPanic()
	}()

	go func() {
		fmt.Println("anon")
	}()

	go func() {}()

}

func (p *Test) Empty() {

}

func (p *Test) Foo() {
	defer fmt.Println("Foo")
}

func (p *Test) FooOK() {
	defer gocommon.LogOnPanic()
}

func (p *Test) FooNotDefer() {
	gocommon.LogOnPanic()
}

func Empty() {

}

func Bar() {
	defer fmt.Println("Bar")
}

func BarOK() {
	defer gocommon.LogOnPanic()
}

func BarNotDefer() {
	gocommon.LogOnPanic()
}
