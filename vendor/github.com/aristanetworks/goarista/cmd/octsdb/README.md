# octsdb

This is a client for the OpenConfig gRPC interface that pushes telemetry to
OpenTSDB.  Non-numerical data isn't supported by OpenTSDB and is silently
dropped.

This tool requires a config file to specify how to map the path of the
notificatons coming out of the OpenConfig gRPC interface onto OpenTSDB
metric names, and how to extract tags from the path.  For example, the
following rule, excerpt from `sampleconfig.json`:

```json
   "metrics": {
      "tempSensor": {
         "path": "/Sysdb/(environment)/temperature/status/tempSensor/(?P<sensor>.+)/((?:maxT|t)emperature)/value"
      },
	  ...
```

Applied to an update for the path
`/Sysdb/environment/temperature/status/tempSensor/TempSensor1/temperature/value`
will lead to the metric name `environment.temperature` and tags `sensor=TempSensor1`.

Basically, un-named groups are used to make up the metric name, and named
groups are used to extract (optional) tags.

## Usage

See the `-help` output, but here's an example to push all the metrics defined
in the sample config file:
```
octsdb -addrs <switch-hostname>:6042 -config sampleconfig.json -text | nc <tsd-hostname> 4242
```
