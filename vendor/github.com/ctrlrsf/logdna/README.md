# logdna

A go library for sending logs to LogDNA via their ingest API.

It works but is still under development and subject to change.

## Installing

```
go get github.com/ctrlrsf/logdna/...
```

## Using library

Godoc available at [https://godoc.org/github.com/ctrlrsf/logdna]()

See source code of `logdna-stdin` command in this repo for an example.

## Using logdna-stdin command

`logdna-stdin` needs to read your LogDNA API key from environment variable `LOGDNA_API_KEY` so make sure its set:

```
export LOGDNA_API_KEY=xyz
```

Usage:

```
Usage of logdna-stdin:
  -hostname string
        hostname you want logs to appear from in LogDNA viewer
  -log-file-name string
        log file or app name you want logs to appear as in LogDNA viewer
```

To send logs to LogDNA just pipe anything that writes to stdout to the logdna-stdin command.

```
$ some_command | logdna-stdin --hostname test.host --log-file-name test.log
```

From LogDNA viewer you'll now see output from `some_command` in log file `test.log` and can filter by host `test.host`.
