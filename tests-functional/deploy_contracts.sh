#!/bin/sh

set -e

echo "Deploying SNT and Communities contracts..."

# Deploy SNT contracts
echo "Deploying SNT contracts..."
/app/clone_and_run.sh status-im status-network-token-v2 script Deploy.s.sol $DEPLOYER_PRIVATE_KEY $DEPLOYER_ADDRESS

# Extract SNT addresses from broadcast output
SNT_BROADCAST_FILE="/app/status-network-token-v2/broadcast/Deploy.s.sol/31337/run-latest.json"
if [ -f "$SNT_BROADCAST_FILE" ]; then
    # Parse the broadcast JSON to extract contract addresses using Python
    python3 << 'PYTHON_SCRIPT' > $CONTRACTS_PATH/snt_addresses.json
import json

with open("/app/status-network-token-v2/broadcast/Deploy.s.sol/31337/run-latest.json", "r") as f:
    data = json.load(f)

returns = data.get("returns", {})
snt_address = None
controller_address = None

for key, value in returns.items():
    if value.get("internal_type") == "contract SNTV2":
        snt_address = value.get("value")
    elif value.get("internal_type") == "contract SNTTokenController":
        controller_address = value.get("value")

result = {
    "snt": snt_address,
    "controller": controller_address
}

print(json.dumps(result, indent=2))
PYTHON_SCRIPT

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
    # Save the entire returns object from the broadcast file
    cat "$COMMUNITIES_BROADCAST_FILE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(json.dumps(data.get('returns', {}), indent=2))" > $CONTRACTS_PATH/communities_addresses.json

    echo "Communities contracts deployed successfully"
    echo "Addresses saved to $CONTRACTS_PATH/communities_addresses.json"
else
    echo "Error: Communities broadcast file not found!"
    exit 1
fi

echo "All contracts deployed successfully!"
