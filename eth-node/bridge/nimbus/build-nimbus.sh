#!/usr/bin/env bash

# Pre-requisites: Git, Nix

set -e

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)

# NOTE: To use a local Nimbus repository, uncomment and edit the following line
#nimbus_dir=~/src/github.com/status-im/nimbus

target_dir="${GIT_ROOT}/vendor/github.com/status-im/status-go/eth-node/bridge/nimbus"

if [ -z "$nimbus_dir" ]; then
  # The git ref of Nimbus to fetch and build. This should represent a commit SHA or a tag, for reproducible builds
  nimbus_ref='master' # TODO: Use a tag once

  nimbus_src='https://github.com/status-im/nimbus/'
  nimbus_dir="${GIT_ROOT}/vendor/github.com/status-im/nimbus"

  trap "rm -rf $nimbus_dir" ERR INT QUIT

  # Clone nimbus repo into vendor directory, if necessary
  if [ -d "$nimbus_dir" ]; then
    cd $nimbus_dir && git reset --hard $nimbus_ref; cd -
  else
    # List fetched from vendorDeps array in https://github.com/status-im/nimbus/blob/master/nix/nimbus-wrappers.nix#L9-L12
    vendor_paths=( nim-chronicles nim-faststreams nim-json-serialization nim-chronos nim-eth nim-json nim-metrics nim-secp256k1 nim-serialization nim-stew nim-stint nimcrypto )
    vendor_path_opts="${vendor_paths[@]/#/--recurse-submodules=vendor/}"
    git clone $nimbus_src --progress ${vendor_path_opts} --depth 1 -j8 -b $nimbus_ref $nimbus_dir
  fi
fi

# Build Nimbus wrappers and copy them into the Nimbus bridge in status-eth-node
build_dir=$(cd $nimbus_dir && nix-build --pure --no-out-link -A wrappers)
# Ideally we'd use the static version of the Nimbus library (.a),
# however that causes link errors due to duplicate symbols:
# ${target_dir}/libnimbus.a(secp256k1.c.o): In function `secp256k1_context_create':
# (.text+0xca80): multiple definition of `secp256k1_context_create'
# /tmp/go-link-476687730/000014.o:${GIT_ROOT}/vendor/github.com/ethereum/go-ethereum/crypto/secp256k1/./libsecp256k1/src/secp256k1.c:56: first defined here
rm -f ${target_dir}/libnimbus.*
mkdir -p ${target_dir}
cp -f ${build_dir}/include/* ${build_dir}/lib/libnimbus.so \
      ${target_dir}/
