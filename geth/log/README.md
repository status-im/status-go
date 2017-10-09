# log [![GoDoc](https://godoc.org/github.com/status-im/status-go/geth/log?status.png)](https://godoc.org/github.com/status-im/status-go/geth/log)
Package log implements structured logging for status-go.

Download:
```shell
go get github.com/status-im/status-go/geth/log
```

* * *
Package log implements Metrics for status-go.

This Metric interface provides a single method where Entry objects which contains 
a map of key-value pairs and timestamp details can be sent as means to record any
series of details associated with a log level and message.

The package exposes a method to initialize a Metric object to be used as the central 
receiving point for all entries.

## Initialization
By default package log has no default Metric set, this is because it intentionally does not 
by default desire you to use it at package level, but if the does arise, then the following should
suffice to achieve this.

- Use a Metric instance with displays to stdout.

```
import "github.com/status-im/status-go/geth/log/custom"

log.Init(log.New(custom.FlatDisplay(os.Stdout)))
```

- Use a Metric instance with a `YellowAlert` level

```
import "github.com/status-im/status-go/geth/log/custom"

log.Init(log.FilterLevel(log.YellowAlert, custom.FlatDisplay(os.Stdout)))
```

- Use a Metric instance with a YellowAlert level, display on stdout and writes every 100 entries to a file in json.

```
import "github.com/status-im/status-go/geth/log/custom"
import "github.com/status-im/status-go/geth/log/jsonfile"

log.Init(log.FilterLevel(log.YellowAlert, custom.FlatDisplay(os.Stdout), jsonfile.JSON("/path/to/geth.log", 100, 2 * time.Second)))
```

## Usage
First, import package into your code:

```
import "github.com/status-im/status-go/geth/log"
```

Then simply use `Info/Error/YellowAlert/etc` functions to log at desired level:

```
log.Send(log.Info("Info message"))
log.Send(log.Debug("Debug message"))
log.Send(log.Error("Error message"))
```

Slightly more complicated logging:

```
log.Send(log.YellowAlert("abnormal conn rate").
With("rate", curRate).
With("low", lowRate).
With("high", highRate))
```

```
log.Send(log.YellowAlert("abnormal conn rate").
WithField(log.Field{
    "rate": curRate,
    "low": lowRate,
    "high": highRate,
}))
```

Logging with Trace data:

```
tr := log.NewTrace("whisper.SendMessage")

defer log.Send(log.Info("Sending whisper messages through connection").
WithTrace(tr.End())
```
