#!/bin/bash

version=10  # Excluding `v`

echo "Bumping status-go module version to v${version}"

# Update template files
sed -i '' "s|github.com/status-im/status-go/|github.com/status-im/status-go/v${version}/|g" cmd/generate_handlers/generate_handlers_template.txt
sed -i '' "s|github.com/status-im/status-go/|github.com/status-im/status-go/v${version}/|g" cmd/status-backend/server/parse-api/endpoints_template.txt
sed -i '' "s|github.com/status-im/status-go/|github.com/status-im/status-go/v${version}/|g" cmd/library/const.go

# Manually update certain files
sed -i '' "s|github.com/status-im/status-go/rpc|github.com/status-im/status-go/v${version}/rpc|g" Makefile
sed -i '' "s|github.com/status-im/status-go/protocol|github.com/status-im/status-go/v${version}/protocol|g" _assets/scripts/run_unit_tests.sh
sed -i '' "s|github.com/status-im/status-go/eth-node|github.com/status-im/status-go/v${version}/eth-node|g" eth-node/Makefile

# Run upgrade tool
mod upgrade -tag ${version}

# Fix "/v3/v3" bug
find . -name "*.go" -exec sed -i '' "s/\/v${version}\/v${version}\//\/v${version}\//g" {} +
