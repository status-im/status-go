// command otto runs JavaScript from a file, opens a repl, or does both.
package main

import (
	"flag"
	"io"
	"io/ioutil"

	"github.com/status-im/ottoext/loop"
	"github.com/status-im/ottoext/loop/looptask"
	erepl "github.com/status-im/ottoext/repl"
	"github.com/robertkrimen/otto"
	"github.com/robertkrimen/otto/repl"

	"github.com/status-im/ottoext/fetch"
	"github.com/status-im/ottoext/process"
	"github.com/status-im/ottoext/promise"
	"github.com/status-im/ottoext/timers"
)

var (
	openRepl = flag.Bool("repl", false, "Always open a REPL, even if a file is provided.")
	debugger = flag.Bool("debugger", false, "Attach REPL-based debugger.")
)

func main() {
	flag.Parse()

	vm := otto.New()

	if *debugger {
		vm.SetDebuggerHandler(repl.DebuggerHandler)
	}

	l := loop.New(vm)

	if err := timers.Define(vm, l); err != nil {
		panic(err)
	}
	if err := promise.Define(vm, l); err != nil {
		panic(err)
	}
	if err := fetch.Define(vm, l); err != nil {
		panic(err)
	}
	if err := process.Define(vm, flag.Args()); err != nil {
		panic(err)
	}

	blockingTask := looptask.NewEvalTask("")

	if len(flag.Args()) == 0 || *openRepl {
		l.Add(blockingTask)
	}

	if len(flag.Args()) > 0 {
		d, err := ioutil.ReadFile(flag.Arg(0))
		if err != nil {
			panic(err)
		}

		// this is a very cheap way of "supporting" shebang lines
		if d[0] == '#' {
			d = []byte("// " + string(d))
		}

		s, err := vm.Compile(flag.Arg(0), string(d))
		if err != nil {
			panic(err)
		}

		if err := l.Eval(s); err != nil {
			panic(err)
		}
	}

	if len(flag.Args()) == 0 || *openRepl {
		go func() {
			if err := erepl.Run(l); err != nil && err != io.EOF {
				panic(err)
			}

			l.Ready(blockingTask)
		}()
	}

	if err := l.Run(); err != nil {
		panic(err)
	}
}
