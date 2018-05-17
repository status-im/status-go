# statusd_node_exporter

The `statusd_node_exporter` can be use to expose metrics consumed by [Prometheus](https://github.com/prometheus/prometheus).

## Usage

```
cd $STATUS_GO_HOME/statusd_node_exporter && \
go build && \
./statusd_node_exporter -ipc ../wnode-status-data/geth.ipc -filter="whisper_*" -filter="les_*"
```
