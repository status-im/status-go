#!/usr/bin/env bash

set -ex

if [ -z "$GETH_VERSION" ]; then
    echo "GETH_VERSION is undefined"
    exit 1
fi

pushd $GOPATH/src/github.com/status-im/go-ethereum
git fetch
git checkout -b patched/$GETH_VERSION origin/patched/$GETH_VERSION || git checkout patched/$GETH_VERSION
git pull
popd

cp -R ./_assets/patches $GOPATH/src/github.com/status-im/go-ethereum

pushd $GOPATH/src/github.com/status-im/go-ethereum
./patches/patcher -b .
rm -r ./patches

git commit -am "add new patches to $GETH_VERSION"
git tag -d $GETH_VERSION
git tag -a $GETH_VERSION -m "Patched release $GETH_VERSION"

echo "Now, go to $GOPATH/src/github.com/status-im/go-ethereum and check if the latest comment in patched/$GETH_VERSION branch is correct and if tag $GETH_VERSION is updated. If they both are ok, push changes rememberting to push the tag with -f option as it already exists."
