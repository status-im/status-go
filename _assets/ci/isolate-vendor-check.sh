#!/usr/bin/env bash

# This is a hack to isolate vendor check in a
# separate clean state. Without this, validate-vendor.sh
# doesn't like our workflow with patches.
#
# How it works:
# 1) Stashes all changes and checks out to a temporary branch.
# 2) Reverts all patches and commits changes.
# 3) Runs "dep ensure" and validate-vendor.sh. Saves exit code and message.
# 4) Commits any changes.
# 5) Goes back to previous branch and removes the temporary branch.
# 6) Applies stashed changes.
# 7) Prints the message and exits with the exit code.

# Stash current changes first, apply later before exiting.
hasChanges=0
changes=($(git status --porcelain))
if [ "$changes" ]; then
	git stash
	hasChanges=1
fi

branchName="$(git rev-parse --abbrev-ref HEAD)"

git checkout -b isolated-vendor-check

# Revert all patches.
$(pwd)/_assets/patches/patcher -r
git add .
git commit -m "vendor check - auto"

# Do vendor check.
dep ensure
msg=$("$(pwd)/_assets/ci/validate-vendor.sh")
failed=$?
git add .
git commit -m "vendor check - auto"

# Go back to previous branch, clean and apply stashed.
git checkout "$branchName"
git branch -D isolated-vendor-check
if [ $hasChanges -eq 1 ]; then
	git stash apply
fi

echo $msg
exit $failed
