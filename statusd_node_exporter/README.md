
go build && ./statusd_node_exporter -ipc ../wnode-status-data/geth.ipc -filter="whisper_*" -filter="les_*"
