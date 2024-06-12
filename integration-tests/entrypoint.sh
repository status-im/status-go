#!/bin/sh

git clone https://github.com/$GITHUB_ORG/$GITHUB_REPO
cd $GITHUB_REPO
git submodule deinit --force .
git submodule update --init --recursive

forge build
exec "$@"
