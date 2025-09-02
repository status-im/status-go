#!/bin/zsh

nwaku=/Users/sirotin/Repositories/status/nwaku/build/wakunode2

# PeerID: 16Uiu2HAm3vFYHkGRURyJ6F7bwDyzMLtPEuCg4DU89T7km2u8Fjyb
$nwaku \
      --nat=extip:127.0.0.1 \
      --listen-address=127.0.0.1 \
      --discv5-discovery=true \
      --discv5-enr-auto-update=True \
      --discv5-udp-port=9001 \
      --cluster-id=16 \
      --filter=true \
      --keep-alive=true \
      --lightpush=true \
      --log-level=TRACE \
      --max-connections=18000 \
      --nodekey=03ce9122016be1f80a23df36525103bed1c7c4b9a0ff7605d97553ed8ed96bcf \
      --peer-exchange=false \
      --relay=true \
      --rest-address=127.0.0.1 \
      --rest-port=8645 \
      --rest-admin \
      --shard=32 \
      --shard=64 \
      --tcp-port=60001
