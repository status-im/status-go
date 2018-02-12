Whisper Nodes Cluster
=====================

This is an instruction how to start a local Whisper cluster in Docker with docker-compose.

## Prepare Docker image

First, you need to have an image called `status-go:latest`. To build it, run `make docker-image`.

## Run cluster

To run the cluster with docker-compose, execute:
```
$ docker-compose up -d --scale wnode=3
```

> It may be required to make your node a swarm manager first. If you see an appropriate error, follow the instructions.

This command will start a bootnode and three Whisper nodes. All Whisper nodes use the single bootnode and they should quickly find each other as peers.

To access debug info, run `docker ps`, figure out your host port bound to wnode `8080/tcp` port and execute `curl localhost:$HOST_PORT/debug/vars`.

Of course, you can always use `docker logs` to read logs. By default, `loglevel` is set to `DEBUG`.
