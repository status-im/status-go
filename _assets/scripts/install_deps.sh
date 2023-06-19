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
  case "$OSTYPE" in
  darwin*)  echo "OSX detected: Install either brew or nix package manager and rerun the script" ;; 
  linux*)   echo "LINUX detected: install either pacman or nix and rerun the script" ;;
  bsd*)     echo "BSD detected: Install either pacman or nix package manager and rerun the script" ;;
  *)        echo "unknown: $OSTYPE" ;;
esac

  exit 1
fi
