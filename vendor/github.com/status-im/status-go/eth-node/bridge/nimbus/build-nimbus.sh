#!/usr/bin/env bash

# Pre-requisites: Git, Nix

set -e

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)

# NOTE: To use a local Nimbus repository, uncomment and edit the following line
#nimbus_dir=~/src/github.com/status-im/nimbus

target_dir="${GIT_ROOT}/vendor/github.com/status-im/status-go/eth-node/bridge/nimbus"

if [ -z "$nimbus_dir" ]; then
  # The git ref of Nimbus to fetch and build. This should represent a commit SHA or a tag, for reproducible builds
  nimbus_ref='feature/android-api' # TODO: Use a tag once

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
build_dir=$(nix-build --pure --no-out-link -A wrappers-native $nimbus_dir/nix/default.nix)
rm -f ${target_dir}/libnimbus.*
mkdir -p ${target_dir}
cp -f ${build_dir}/include/* ${build_dir}/lib/libnimbus.so \
      ${target_dir}/
chmod +w ${target_dir}/libnimbus.{so,h}
