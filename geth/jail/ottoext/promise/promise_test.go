package promise

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

func TestResolve(t *testing.T) {
	vm := otto.New()
	l := loop.New(vm)

	if err := Define(vm, l); err != nil {
		panic(err)
	}

	return

	must(l.EvalAndRun(`
		var p = new Promise(function(resolve, reject) {
			setTimeout(function() {
				resolve('good');
			}, 10);
		});

		p.then(function(d) {
			if (d !== 'good') {
				throw new Error('invalid resolution');
			}
		});

		p.catch(function(err) {
			throw err;
		});
	`))
}

func TestReject(t *testing.T) {
	vm := otto.New()
	l := loop.New(vm)

	if err := Define(vm, l); err != nil {
		panic(err)
	}

	must(l.EvalAndRun(`
		var p = new Promise(function(resolve, reject) {
			setTimeout(function() {
				reject('bad');
			}, 10);
		});

		p.catch(function(err) {
			if (err !== 'bad') {
				throw new Error('invalid rejection');
			}
		});
	`))
}
