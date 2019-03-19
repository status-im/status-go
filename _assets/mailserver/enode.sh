#!/bin/bash

DOCKER_CONTAINER="${DOCKER_CONTAINER:-status-mailserver-node}"

ENODE=$(docker exec -ti $DOCKER_CONTAINER bash -c "echo '{\"jsonrpc\":\"2.0\",\"method\":\"admin_nodeInfo\",\"id\":1}' | socat - UNIX-CONNECT:/geth.ipc | jq '.result.enode'")
echo "Mailserver enode: ${ENODE}"
