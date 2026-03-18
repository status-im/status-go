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
    REGISTRY_ADDR=$(grep -A1 '"contractName": "ENSRegistry"' "$ENS_BROADCAST_FILE" | grep '"contractAddress"' | head -1 | sed 's/.*"contractAddress": "\([^"]*\)".*/\1/')
    RESOLVER_ADDR=$(grep -A1 '"contractName": "PublicResolver"' "$ENS_BROADCAST_FILE" | grep '"contractAddress"' | head -1 | sed 's/.*"contractAddress": "\([^"]*\)".*/\1/')
    TOKEN_ADDR=$(grep -A1 '"contractName": "MiniMeToken"' "$ENS_BROADCAST_FILE" | grep '"contractAddress"' | head -1 | sed 's/.*"contractAddress": "\([^"]*\)".*/\1/')
    REGISTRAR_ADDR=$(grep -A1 '"contractName": "UsernameRegistrar"' "$ENS_BROADCAST_FILE" | grep '"contractAddress"' | head -1 | sed 's/.*"contractAddress": "\([^"]*\)".*/\1/')

    echo "ENS Registry (deployed): $REGISTRY_ADDR"
    echo "ENS Resolver: $RESOLVER_ADDR"
    echo "ENS Token: $TOKEN_ADDR"
    echo "ENS Registrar: $REGISTRAR_ADDR"

    # Domain setup on deployed registry — registrar and resolver reference it via immutables
    echo "Setting up stateofus.eth domain..."
    ROOT_NODE="0x0000000000000000000000000000000000000000000000000000000000000000"
    ETH_NAMEHASH=$(cast namehash "eth")
    ETH_LABELHASH=$(cast keccak "eth")
    STATEOFUS_LABELHASH=$(cast keccak "stateofus")

    cast send $REGISTRY_ADDR "setSubnodeOwner(bytes32,bytes32,address)" \
        $ROOT_NODE $ETH_LABELHASH $DEPLOYER_ADDRESS \
        --rpc-url $ANVIL_URL --private-key $DEPLOYER_PRIVATE_KEY

    cast send $REGISTRY_ADDR "setSubnodeOwner(bytes32,bytes32,address)" \
        $ETH_NAMEHASH $STATEOFUS_LABELHASH $REGISTRAR_ADDR \
        --rpc-url $ANVIL_URL --private-key $DEPLOYER_PRIVATE_KEY

    # Activate registrar (required before it accepts registrations)
    echo "Activating registrar..."
    cast send $REGISTRAR_ADDR "activate(uint256)" 1000000000000000000 \
        --rpc-url $ANVIL_URL --private-key $DEPLOYER_PRIVATE_KEY

    # Go code (go-ens, resolver/address.go) queries the well-known ENS registry.
    # Copy deployed registry code + storage to well-known address so Go can read it.
    WELL_KNOWN_REGISTRY="0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e"
    echo "Syncing deployed registry to well-known address..."
    /app/sync_ens_registry.sh $REGISTRY_ADDR $WELL_KNOWN_REGISTRY $ANVIL_URL

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
