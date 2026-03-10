#!/bin/sh

set -e

echo "Deploying SNT and Communities contracts..."

# Deploy SNT contracts
echo "Deploying SNT contracts..."
/app/clone_and_run.sh status-im status-network-token-v2 script Deploy.s.sol $DEPLOYER_PRIVATE_KEY $DEPLOYER_ADDRESS

# Extract SNT addresses from broadcast output
SNT_BROADCAST_FILE="/app/status-network-token-v2/broadcast/Deploy.s.sol/31337/run-latest.json"
if [ -f "$SNT_BROADCAST_FILE" ]; then
    # Parse the broadcast JSON to extract contract addresses using grep/sed
    SNT_ADDR=$(grep -A1 '"internal_type": "contract SNTV2"' "$SNT_BROADCAST_FILE" | grep '"value"' | sed 's/.*"value": "\([^"]*\)".*/\1/')
    CONTROLLER_ADDR=$(grep -A1 '"internal_type": "contract SNTTokenController"' "$SNT_BROADCAST_FILE" | grep '"value"' | sed 's/.*"value": "\([^"]*\)".*/\1/')

    cat > $CONTRACTS_PATH/snt_addresses.json << EOF
{
  "snt": "$SNT_ADDR",
  "controller": "$CONTROLLER_ADDR"
}
EOF

    echo "SNT deployed successfully"
    echo "Addresses saved to $CONTRACTS_PATH/snt_addresses.json"
else
    echo "Error: SNT broadcast file not found!"
    exit 1
fi

# Deploy Communities contracts
echo "Deploying Communities contracts..."
/app/clone_and_run.sh status-im communities-contracts script DeployContracts.s.sol $DEPLOYER_PRIVATE_KEY $DEPLOYER_ADDRESS

# Extract Communities addresses from broadcast output
COMMUNITIES_BROADCAST_FILE="/app/communities-contracts/broadcast/DeployContracts.s.sol/31337/run-latest.json"
if [ -f "$COMMUNITIES_BROADCAST_FILE" ]; then
    # Extract returns section using sed
    sed -n '/"returns":/,/^  },$/p' "$COMMUNITIES_BROADCAST_FILE" | sed '1s/"returns"://' | sed '$s/,$//' > $CONTRACTS_PATH/communities_addresses.json

    echo "Communities contracts deployed successfully"
    echo "Addresses saved to $CONTRACTS_PATH/communities_addresses.json"
else
    echo "Error: Communities broadcast file not found!"
    exit 1
fi

# Deploy ENS contracts
echo "Deploying ENS contracts..."
/app/clone_and_run.sh status-im ens-usernames script Deploy.s.sol $DEPLOYER_PRIVATE_KEY $DEPLOYER_ADDRESS

ENS_BROADCAST_FILE="/app/ens-usernames/broadcast/Deploy.s.sol/31337/run-latest.json"
if [ -f "$ENS_BROADCAST_FILE" ]; then
    REGISTRY_ADDR=$(grep -B5 '"contractName": "ENSRegistry"' "$ENS_BROADCAST_FILE" | grep '"contractAddress"' | head -1 | sed 's/.*"contractAddress": "\([^"]*\)".*/\1/')
    RESOLVER_ADDR=$(grep -B5 '"contractName": "PublicResolver"' "$ENS_BROADCAST_FILE" | grep '"contractAddress"' | head -1 | sed 's/.*"contractAddress": "\([^"]*\)".*/\1/')
    TOKEN_ADDR=$(grep -B5 '"contractName": "MiniMeToken"' "$ENS_BROADCAST_FILE" | grep '"contractAddress"' | head -1 | sed 's/.*"contractAddress": "\([^"]*\)".*/\1/')
    REGISTRAR_ADDR=$(grep -B5 '"contractName": "UsernameRegistrar"' "$ENS_BROADCAST_FILE" | grep '"contractAddress"' | head -1 | sed 's/.*"contractAddress": "\([^"]*\)".*/\1/')

    echo "ENS Registry (deployed): $REGISTRY_ADDR"
    echo "ENS Resolver: $RESOLVER_ADDR"
    echo "ENS Token: $TOKEN_ADDR"
    echo "ENS Registrar: $REGISTRAR_ADDR"

    # go-ens library and status-go resolver hardcode this address
    WELL_KNOWN_REGISTRY="0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e"

    # Copy registry bytecode to well-known address
    echo "Copying ENS registry to well-known address..."
    REGISTRY_CODE=$(cast code $REGISTRY_ADDR --rpc-url $ANVIL_URL)
    cast rpc anvil_setCode $WELL_KNOWN_REGISTRY $REGISTRY_CODE --rpc-url $ANVIL_URL

    # Copy storage for root, eth, stateofus.eth nodes from deployed to well-known
    ROOT_NODE="0x0000000000000000000000000000000000000000000000000000000000000000"
    ETH_NAMEHASH=$(cast namehash "eth")
    STATEOFUS_NAMEHASH=$(cast namehash "stateofus.eth")

    for NODE in $ROOT_NODE $ETH_NAMEHASH $STATEOFUS_NAMEHASH; do
        BASE_SLOT=$(cast keccak $(cast abi-encode "f(bytes32,uint256)" $NODE 0))
        for OFFSET in 0 1 2; do
            SLOT=$(cast to-hex $(($(cast to-dec $BASE_SLOT) + $OFFSET)) 2>/dev/null || python3 -c "print(hex(int('$BASE_SLOT', 16) + $OFFSET))")
            VALUE=$(cast storage $REGISTRY_ADDR $SLOT --rpc-url $ANVIL_URL)
            cast rpc anvil_setStorageAt $WELL_KNOWN_REGISTRY $SLOT $VALUE --rpc-url $ANVIL_URL
        done
    done

    # Set up stateofus.eth domain on well-known registry (not deployed),
    # because Go resolves registrar via OwnerOf("stateofus.eth") on well-known
    echo "Setting up stateofus.eth domain on well-known registry..."
    ETH_LABELHASH=$(cast keccak "eth")
    STATEOFUS_LABELHASH=$(cast keccak "stateofus")

    cast send $WELL_KNOWN_REGISTRY "setSubnodeOwner(bytes32,bytes32,address)" \
        $ROOT_NODE $ETH_LABELHASH $DEPLOYER_ADDRESS \
        --rpc-url $ANVIL_URL --private-key $DEPLOYER_PRIVATE_KEY

    cast send $WELL_KNOWN_REGISTRY "setSubnodeOwner(bytes32,bytes32,address)" \
        $ETH_NAMEHASH $STATEOFUS_LABELHASH $REGISTRAR_ADDR \
        --rpc-url $ANVIL_URL --private-key $DEPLOYER_PRIVATE_KEY

    # Patch registrar (slot 1) and resolver (slot 0) to use well-known registry,
    # so register() updates the registry that Go actually queries
    echo "Patching registrar and resolver to use well-known registry..."
    WELL_KNOWN_PADDED=$(echo $WELL_KNOWN_REGISTRY | sed 's/0x//' | tr '[:upper:]' '[:lower:]')
    WELL_KNOWN_SLOT_VALUE="0x000000000000000000000000${WELL_KNOWN_PADDED}"

    cast rpc anvil_setStorageAt $REGISTRAR_ADDR \
        "0x0000000000000000000000000000000000000000000000000000000000000001" \
        $WELL_KNOWN_SLOT_VALUE --rpc-url $ANVIL_URL

    cast rpc anvil_setStorageAt $RESOLVER_ADDR \
        "0x0000000000000000000000000000000000000000000000000000000000000000" \
        $WELL_KNOWN_SLOT_VALUE --rpc-url $ANVIL_URL

    # Activate registrar (required before it accepts registrations)
    echo "Activating registrar..."
    cast send $REGISTRAR_ADDR "activate(uint256)" 1000000000000000000 \
        --rpc-url $ANVIL_URL --private-key $DEPLOYER_PRIVATE_KEY

    cat > $CONTRACTS_PATH/ens_addresses.json << EOF
{
  "registry": "$REGISTRY_ADDR",
  "resolver": "$RESOLVER_ADDR",
  "token": "$TOKEN_ADDR",
  "registrar": "$REGISTRAR_ADDR"
}
EOF

    echo "ENS contracts deployed and configured successfully"
else
    echo "Error: ENS broadcast file not found!"
    exit 1
fi

echo "All contracts deployed successfully!"
