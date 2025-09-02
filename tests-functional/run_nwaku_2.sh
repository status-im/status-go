#!/bin/zsh

nwaku=/Users/sirotin/Repositories/status/nwaku/build/wakunode2

# PeerID: 16Uiu2HAm5TrV8hjUQaqJ8VaHC1EL3EX8RdqP1gTrLxSCecWVN3xc
$nwaku \
      --discv5-discovery=true \
      --discv5-enr-auto-update=True \
      --discv5-udp-port=9002 \
      --cluster-id=16 \
      --filter=true \
      --keep-alive=true \
      --lightpush=true \
      --log-level=DEBUG \
      --max-connections=18000 \
      --nodekey=c71da918e785259ebf13a3cf2f18d6f7c1329983ca17636f473c9e80ae5dcafc \
      --peer-exchange=false \
      --relay=true \
      --rest-address=127.0.0.1 \
      --rest-port=8646 \
      --rest-admin \
      --shard=32 \
      --shard=64 \
      --tcp-port=60002 \
      --discv5-bootstrap-node=enr:-LO4QPd9pwbTsQdQnajBXvvYlCYZ_4jqehC5sbFN5hK6vBDiNh_q7CazaXIAhXgqwLshUVSAmuKmgRzRpCiU90tRAhIEgmlkgnY0gmlwhFr7Ho-KbXVsdGlhZGRyc4oACARa-x6PBuphgnJzhwAQAgAgAECJc2VjcDI1NmsxoQJ-L7RtImvRuvgn_pQnJsag8tbaqPZ7mBM4Z1scEreWGoN0Y3CC6mGDdWRwgiMphXdha3UyDQ
