#!/bin/zsh

waku=/Users/sirotin/Repositories/status/go-waku/build/waku

# PeerID: 16Uiu2HAm4Gk5X5JmcL3hW6wMEc67dkMWrUEnvEjdrxLcQy2cNJfF
$waku \
      --listen-address=127.0.0.1 \
      --discv5-discovery=true \
      --discv5-enr-auto-update=True \
      --discv5-udp-port=9005 \
      --cluster-id=16 \
      --filter=true \
      --lightpush=true \
      --log-level=DEBUG \
      --max-connections=18000 \
      --nodekey=032f6df3231a0820e65d58c1728619ff247dcc8e43daf1a5e42368d9bafc8f9e \
      --peer-exchange=false \
      --relay=true \
      --rest \
      --rest-address=127.0.0.1 \
      --rest-port=8649 \
      --rest-admin \
      --pubsub-topic=/waku/2/rs/16/32 \
      --pubsub-topic=/waku/2/rs/16/64 \
      --tcp-port=60005 \
      --discv5-bootstrap-node=enr:-LO4QMyeW03PjuZwMaVtJe9C-Tk-UPYE1jfdefDFyuR3eb89O8dTawtkwLoicuAQSxZ_OiHcIYELBkkmqGPKkDuadowCgmlkgnY0gmlwhH8AAAGKbXVsdGlhZGRyc4oACATAqClFBuphgnJzhwAQAgAgAECJc2VjcDI1NmsxoQJ-L7RtImvRuvgn_pQnJsag8tbaqPZ7mBM4Z1scEreWGoN0Y3CC6mGDdWRwgiMphXdha3UyDQ