#!/bin/bash

version=3
oldVersion=$((version - 1))

echo "Upgrading from version ${oldVersion} to ${version}"

# Remove sub-modules
#rm extkeys/go.sum
#rm extkeys/go.mod
rm exchanges/go.sum
rm exchanges/go.mod

# Update template files
sed -i '' "s|github.com/status-im/status-go/v${oldVersion}|github.com/status-im/status-go/v${version}|g" cmd/generate_handlers/generate_handlers_template.txt
sed -i '' "s|github.com/status-im/status-go/v${oldVersion}|github.com/status-im/status-go/v${version}|g" cmd/status-backend/server/parse-api/endpoints_template.txt

# Run upgrade tool
mod upgrade -tag ${version}

# Fix "/v3/v3" bug
find . -name "*.go" -exec sed -i '' "s/\/v${version}\/v${version}/\/v${version}/g" {} +

# Fallback "/v3/extkeys" to "/extkeys"
sed -i '' "s|github.com/status-im/status-go/v${oldVersion}|github.com/status-im/status-go/v${version}|g"