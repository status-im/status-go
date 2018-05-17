# statusd_node_exporter

## Usage

```
cd $STATUS_GO_HOME/statusd_node_exporter && \
go build && \
./statusd_node_exporter -ipc ../wnode-status-data/geth.ipc -filter="whisper_*" -filter="les_*"
```
