#!/bin/zsh

waku=/Users/sirotin/Repositories/status/go-waku/build/waku

# PeerID: 16Uiu2HAmQAuAw9ANRfXu7MvmJoEheAk4TfVMYHq737S436ipoHYc
$waku \
      --listen-address=127.0.0.1 \
      --discv5-discovery=true \
      --discv5-enr-auto-update=True \
      --discv5-udp-port=9004 \
      --cluster-id=16 \
      --filter=true \
      --lightpush=true \
      --log-level=DEBUG \
      --max-connections=18000 \
      --nodekey=c497a860343da5e39c9f9635cd444c60e6e65259d063c58028762c606a992ce9 \
      --peer-exchange=false \
      --relay=true \
      --rest \
      --rest-address=127.0.0.1 \
      --rest-port=8648 \
      --rest-admin \
      --pubsub-topic=/waku/2/rs/16/32 \
      --pubsub-topic=/waku/2/rs/16/64 \
      --tcp-port=60004 \
      --discv5-bootstrap-node=enr:-LO4QMyeW03PjuZwMaVtJe9C-Tk-UPYE1jfdefDFyuR3eb89O8dTawtkwLoicuAQSxZ_OiHcIYELBkkmqGPKkDuadowCgmlkgnY0gmlwhH8AAAGKbXVsdGlhZGRyc4oACATAqClFBuphgnJzhwAQAgAgAECJc2VjcDI1NmsxoQJ-L7RtImvRuvgn_pQnJsag8tbaqPZ7mBM4Z1scEreWGoN0Y3CC6mGDdWRwgiMphXdha3UyDQ
