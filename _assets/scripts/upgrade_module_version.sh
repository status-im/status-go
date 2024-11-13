#!/bin/bash

version=3
oldVersion=$((version - 1))

echo "version=${version}"
echo "oldVersion=${oldVersion}"

rm extkeys/go.sum
rm extkeys/go.mod
rm exchanges/go.sum
rm exchanges/go.mod

sed -i '' "s|github.com/status-im/status-go/v${oldVersion}|github.com/status-im/status-go/v${version}|g" cmd/generate_handlers/generate_handlers_template.txt
sed -i '' "s|github.com/status-im/status-go/v${oldVersion}|github.com/status-im/status-go/v${version}|g" cmd/status-backend/server/parse-api/endpoints_template.txt

#make generate

mod upgrade -tag ${version}

find . -name "*.go" -exec sed -i '' "s/\/v${version}\/v${version}/\/v${version}/g" {} +