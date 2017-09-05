# expvars
--

Expvars package is a helper package to use "exported vars" metrics in `status-go`.

Expvar is a wrapper around stdlib [expvar](https://golang.org/pkg/expvar/) package, which exposes http endpoint `/debug/vars` with JSON containing exported vars.

This JSON can be later polled by any metrics/monitoring tool for further analysis/display.

## Usage

First, define variable of function in `expvars.go` for easy usage from other packages:

## Creating new var
### Simple variables:

```go
var Peers = expvar.NewInt("Peers")
```

### Functions:
You may also define more complex metrics using functions:

```go
func goroutines() interface{} {
	return runtime.NumGoroutine()
}
...
expvar.Publish("Goroutines", expvar.Func(goroutines))
```
## Using new var
Second, import package:

```go
import "github.com/status-im/status-go/helpers/expvars"
```

then, use exported variable and change its value:

```go
expvars.Peers.Add(1)
```
And that's it. Exported field `"Peers"` with actualized value will appear in JSON returned by `/debug/vars` endpoint.

Note: you may define new vars in any package, but it would be nice to keep all exported vars in one placce.

# Monitoring
Setting up the monitoring/metrics stack is too expensive for debug sessions or even impossible for mobile environment.

There is a tool for easy zero-cost expvar monitoring, [expvarmon](https://github.com/divan/expvarmon). Here is an example output for `statusd`:

![](https://i.imgur.com/oz11bmT.png)

Command to use:

```bash
expvarmon -ports 10000 -vars Goroutines,duration:Uptime,Cells,Peers,mem:memstats.Alloc,mem:memstats.Sys,mem:memstats.HeapAlloc,mem:memstats.HeapInuse,duration:memstats.PauseNs,duration:memstats.PauseTotalNs -i 500ms
```