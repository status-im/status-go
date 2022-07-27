#!/usr/bin/env bash

if [[ -x $(command -v apt-get) ]]; then
  apt-get install -y protobuf-compiler jq
elif [[ -x $(command -v pacman) ]]; then
  pacman -Sy protobuf jq --noconfirm
elif [[ -x $(command -v brew) ]]; then
  brew install protobuf jq
elif [[ -x $(command -v nix-env) ]]; then
  nix-env -iA nixos.protobuf3_17
else
  echo "ERROR: No known package manager found!" >&2
  exit 1
fi
