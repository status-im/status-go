package timers

import (
	"testing"

	"github.com/robertkrimen/otto"

	"github.com/status-im/status-go/geth/jail/ottoext/loop"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func TestSetTimeout(t *testing.T) {
	vm := otto.New()
	l := loop.New(vm)

	if err := Define(vm, l); err != nil {
		panic(err)
	}

	must(l.EvalAndRun(`setTimeout(function(n) {
		if (Date.now() - n < 50) {
			throw new Error('timeout was called too soon');
		}
	}, 50, Date.now());`))
}

func TestClearTimeout(t *testing.T) {
	vm := otto.New()
	l := loop.New(vm)

	if err := Define(vm, l); err != nil {
		panic(err)
	}

	must(l.EvalAndRun(`clearTimeout(setTimeout(function() {
		throw new Error('should never run');
	}, 50));`))
}

func TestSetInterval(t *testing.T) {
	vm := otto.New()
	l := loop.New(vm)

	if err := Define(vm, l); err != nil {
		panic(err)
	}

	must(l.EvalAndRun(`
		var c = 0;
		var iv = setInterval(function() {
			if (c++ === 1) {
				clearInterval(iv);
			}
		}, 50);
	`))
}

func TestClearIntervalImmediately(t *testing.T) {
	vm := otto.New()
	l := loop.New(vm)

	if err := Define(vm, l); err != nil {
		panic(err)
	}

	must(l.EvalAndRun(`clearInterval(setInterval(function() {
		throw new Error('should never run');
	}, 50));`))
}
