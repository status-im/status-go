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

echo "All contracts deployed successfully!"
