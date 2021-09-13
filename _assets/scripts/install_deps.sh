#!/usr/bin/env bash

if [[ -x $(command -v apt) ]]; then
  apt install -y protobuf-compiler jq
elif [[ -x $(command -v pacman) ]]; then
  pacman -Sy protobuf jq --noconfirm
elif [[ -x $(command -v brew) ]]; then
  brew install protobuf jq
elif [[ -x $(command -v nix-env) ]]; then
  nix-env -iA nixos.protobuf3_13
else
  echo "ERROR: No known package manager found!" >&2
  exit 1
fi
