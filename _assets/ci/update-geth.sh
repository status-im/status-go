#!/usr/bin/env bash

# This script updates the go-ethereum dependency, optionally updating the branch if GETH_BRANCH is provided.
# If any changes were made, they will be committed.

# Exit early if any errors are encountered
set -e
if [ ! -z "$GETH_BRANCH" ]; then
	# escape slashes
	GETH_BRANCH=$(echo $GETH_BRANCH | sed 's@\/@\\\/@g')
	# Update go-ethereum contraint branch
	sed -i 'N;N;s@\(\[\[constraint]]\n  name = "github.com\/ethereum\/go-ethereum"\n  branch =\)\(.*\)@\1 '"\"${GETH_BRANCH}\""'@g' Gopkg.toml
fi
dep ensure -v -update github.com/ethereum/go-ethereum
make dep-ensure
git add Gopkg.lock Gopkg.toml vendor/
if $(git diff --cached --quiet); then
	echo "No changes to commit. Geth up to date."
    exit 0
fi
git commit --quiet -m "Updating Geth"
echo "Geth updated."
