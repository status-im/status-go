#!/bin/zsh

nwaku=/Users/sirotin/Repositories/status/nwaku/build/wakunode2

# PeerID: 16Uiu2HAmGqYFBfbKCQmhUagDFVo43dPBoEe4mcp2TThH93AAf55c
$nwaku \
      --listen-address=127.0.0.1 \
      --discv5-discovery=true \
      --discv5-enr-auto-update=True \
      --discv5-udp-port=9003 \
      --cluster-id=16 \
      --filter=true \
      --keep-alive=true \
      --lightpush=true \
      --log-level=DEBUG \
      --max-connections=18000 \
      --nodekey=451ee0adb90a7970dca8a757bca58e9a66c02dcf928e1c55e9b49319a2407ac0 \
      --peer-exchange=true \
      --relay=true \
      --rest-address=127.0.0.1 \
      --rest-port=8647 \
      --rest-admin \
      --shard=32 \
      --shard=64 \
      --tcp-port=60003 \
      --discv5-bootstrap-node=enr:-LO4QMyeW03PjuZwMaVtJe9C-Tk-UPYE1jfdefDFyuR3eb89O8dTawtkwLoicuAQSxZ_OiHcIYELBkkmqGPKkDuadowCgmlkgnY0gmlwhH8AAAGKbXVsdGlhZGRyc4oACATAqClFBuphgnJzhwAQAgAgAECJc2VjcDI1NmsxoQJ-L7RtImvRuvgn_pQnJsag8tbaqPZ7mBM4Z1scEreWGoN0Y3CC6mGDdWRwgiMphXdha3UyDQ