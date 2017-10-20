/*
jail - jailed enviroment for executing JS code.

Package jail implements "jailed" enviroment for executing arbitrary
JavaScript code using Otto JS interpreter (https://github.com/robertkrimen/otto).

Jail create multiple Cells, one cell per status client chat. Each cell runs own
Otto virtual machine and lives forever, but that may change in the future.

  +----------------------------------------------+
  |                     Jail                     |
  +----------------------------------------------+
  +---------+ +---------+ +---------+  +---------+
  |  Cell   | |  Cell   | |  Cell   |  |  Cell   |
  |ChatID 1 | |ChatID 2 | |ChatID 3 |  |ChatID N |
  |+-------+| |+-------+| |+-------+|  |+-------+|
  ||Otto VM|| ||Otto VM|| ||Otto VM||  ||Otto VM||
  |+-------+| |+-------+| |+-------+|  |+-------+|
  || Loop  || || Loop  || || Loop  ||  || Loop  ||
  ++-------++ ++-------++ ++-------++  ++-------++

Cells

Each Cell object embeds *VM from 'jail/vm' for concurrency safe wrapper around
*otto.VM functions. This is important when dealing with setTimeout and Fetch API
functions (see below).

Get and Set

(*VM).Get/Set functions provide transparent and concurrently safe wrappers for
Otto VM Get and Set functions respectively. See Otto documentation for usage examples:
https://godoc.org/github.com/robertkrimen/otto

Call and Run

(*VM).Call/Run functions allows executing arbitrary JS in the cell. They're also
wrappers arount Otto VM functions of the same name. Run accepts raw JS strings for execution,
Call takes a JS function name (defined in VM) and parameters.

Timeouts and intervals support

Default Otto VM interpreter doesn't support setTimeout()/setInterval() JS functions,
because they're not part of ECMA-262 spec, but properties of the window object in browser.
We add support for them using own implementation of Event Loop, heavily based on http://github.com/status-im/ottoext package. See loop/fetch/promise packages under `jail/internal/`.

Each cell starts a new loop in a separate goroutine, registers functions for setTimeout/setInterval
calls and associate them with this loop. All JS code executed as callback to setTimeout/setInterval
will be handled by this loop.

For example, following code:

	cell.Run(`setTimeout(function(){ value = "42" }, 2000);`)

will execute setTimeout and return immidiately, but callback function will
be executed after 2 seconds in the loop that was started upon current cell.

In order to capture response one may use following approach:

	err = cell.Set("__captureResponse", func(val string) otto.Value {
		fmt.Println("Captured response from callback:", val)
		return otto.UndefinedValue()
	})
	cell.Run(`setTimeout(function(){ __captureResponse("OK") }, 2000);`)

Fetch support

Fetch API is implemented in a similar way using the same loop. When Cell is created, corresponding handlers are registered within VM and associated event loop.

Due to asynchronous nature of Fetch API, the following code will return immediately:

	cell.Run(`fetch('http://example.com/').then(function(data) { ... })`)

and callback function in a promise will be executed in a event loop in the background. Thus,
it's user responsibility to register a corresponding callback function before:

	cell.Set("__captureSuccess", func(res otto.Value) { ... })

	cell.Run(`fetch('http://example.com').then(function(r) {
		return r.text()
	}).then(function(data) {
		// user code
		__captureSuccess(data)
	}))

*/
package jail

//go:generate autoreadme -f
