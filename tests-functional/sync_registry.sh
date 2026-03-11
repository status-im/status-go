#!/bin/sh
# Sync ENS registry storage from source to destination address.
# Usage: sync_registry.sh <source_addr> <dest_addr> <rpc_url> [node1 node2 ...]
# If no nodes specified, syncs root, eth, and stateofus.eth nodes.

set -e

SOURCE=$1
DEST=$2
RPC_URL=$3
shift 3

# Default nodes to sync
if [ $# -eq 0 ]; then
    set -- \
        "0x0000000000000000000000000000000000000000000000000000000000000000" \
        "$(cast namehash 'eth')" \
        "$(cast namehash 'stateofus.eth')"
fi

# Copy bytecode
CODE=$(cast code $SOURCE --rpc-url $RPC_URL)
cast rpc anvil_setCode $DEST $CODE --rpc-url $RPC_URL > /dev/null

# Copy storage for each node: owner (base+0), resolver (base+1), ttl (base+2)
for NODE in "$@"; do
    BASE_SLOT=$(cast index bytes32 $NODE 0)
    for OFFSET in 0 1 2; do
        SLOT=$(perl -e "use Math::BigInt; print '0x', Math::BigInt->new('$BASE_SLOT')->badd($OFFSET)->as_hex() =~ s/^0x//r;")
        # Pad to 66 chars (0x + 64 hex)
        SLOT=$(printf "0x%064s" "$(echo $SLOT | sed 's/0x//')")
        VALUE=$(cast storage $SOURCE $SLOT --rpc-url $RPC_URL)
        cast rpc anvil_setStorageAt $DEST $SLOT $VALUE --rpc-url $RPC_URL > /dev/null
    done
done
