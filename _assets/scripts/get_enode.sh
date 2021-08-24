#!/usr/bin/env bash

RPC_ADDR="${RPC_ADDR:-localhost}"
RPC_PORT="${RPC_PORT:-8545}"
# might be provided by parent
if [[ -z "${PUBLIC_IP}" ]]; then
    PUBLIC_IP=$(curl -s https://ipecho.net/plain)
fi
# Necessary for enode address for Status app
MAIL_PASSWORD="${MAIL_PASSWORD:-status-offline-inbox}"

# query local 
RESP_JSON=$(
    curl -sS --retry 3 --retry-all-errors \
        -X POST http://${RPC_ADDR}:${RPC_PORT}/ \
        -H 'Content-type: application/json' \
        -d '{"jsonrpc":"2.0","method":"admin_nodeInfo","params":[],"id":1}'
)
if [[ "$?" -ne 0 ]]; then
    echo "RPC port not up, unable to query enode address!" 1>&2
    exit 1
fi

# extract enode from JSON response
ENODE_RAW=$(echo "${RESP_JSON}" | jq -r '.result.enode')
# drop arguments at the end of enode address
ENODE_CLEAN=$(echo "${ENODE_RAW}" | grep -oP '\Kenode://[^?]+')

# replace localhost with public IP and add mail password
ENODE=$(echo "${ENODE_CLEAN}" | sed \
    -e "s/127.0.0.1/${PUBLIC_IP}/" \
    -e "s/@/:${MAIL_PASSWORD}@/")

if [[ "$1" == "--qr" ]]; then
    if ! [ -x "$(command -v qrencode)" ]; then
      echo 'Install 'qrencode' for enode QR code.' >&2
      exit 0
    fi
    qrencode -t UTF8 "${ENODE}"
else
    echo "${ENODE}"
fi
