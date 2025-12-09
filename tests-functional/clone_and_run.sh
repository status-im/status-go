#!/bin/sh

GITHUB_ORG=$1
GITHUB_REPO=$2
SMART_CONTRACT_DIR=$3
SMART_CONTRACT_FILENAME=$4
PRIVATE_KEY=$5
SENDER_ADDRESS=$6

cd /app

rm -rf $GITHUB_REPO

git clone https://github.com/$GITHUB_ORG/$GITHUB_REPO
cd $GITHUB_REPO
git submodule deinit --force .
git submodule update --init --recursive

forge build
forge script $SMART_CONTRACT_DIR/$SMART_CONTRACT_FILENAME --fork-url=$ETH_RPC_URL --private-key=$PRIVATE_KEY --sender=$SENDER_ADDRESS --broadcast
