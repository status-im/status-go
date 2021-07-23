#!/bin/sh

apk add curl jq

function check() {
  result=$(curl -X POST --data '{"jsonrpc":"2.0","method":"admin_peers", "params": [], "id":1}' http://172.16.238.11:8545 -H "Content-Type: application/json" | jq '.result[] | select(.network.remoteAddress | startswith("172.16.238.10"))')
  test "$result"
}

NEXT_WAIT_TIME=0
until [ $NEXT_WAIT_TIME -eq 5 ] || check; do
    sleep $(( NEXT_WAIT_TIME++ ))
done
[ $NEXT_WAIT_TIME -lt 5 ]
