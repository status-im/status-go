#!/bin/sh

export ANVIL_URL="http://anvil:8545"
export DEPLOYER_ADDRESS="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
export DEPLOYER_PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
export CONTRACTS_PATH="/app/contracts"

echo "Starting Foundry"

echo "Deploying Multicall3"
forge create $CONTRACTS_PATH/Multicall3.sol:Multicall3 --rpc-url $ANVIL_URL --private-key $DEPLOYER_PRIVATE_KEY --broadcast | tee $CONTRACTS_PATH/Multicall3.sol.log

echo "Deploying SNT and Communities contracts"
/app/deploy_contracts.sh

tail -F /dev/null # Keep container running indefinitely
