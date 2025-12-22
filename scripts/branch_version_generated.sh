#!/usr/bin/env bash

set -euo pipefail

# Get latest tag
latest_tag=$(git describe --tags)
branch_name="generated/${latest_tag}"

# Create branch
git checkout -b "${branch_name}"

# Un-gitignore generated files
sed -i '' '/# generated files/,/^$/ s/^/#/' .gitignore

# Generate files
make GO_GENERATE_FAST_RECACHE=true generate

# Remove `-dirty` suffix from the generated version file
sed -i '' 's/-dirty$//' pkg/version/VERSION

# Commit
git add .gitignore
git add .
git commit -m "feat_: version ${latest_tag} with generated files included"
git push origin "${branch_name}"