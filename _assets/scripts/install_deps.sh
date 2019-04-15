#!/bin/bash

if [ -x "$(command -v apt)" ]; then
  echo "apt install -y protobuf-compiler jq"
fi

if [ -x "$(command -v pacman)" ]; then
  pacman -Sy protobuf jq --noconfirm
fi

if [ -x "$(command -v brew)" ]; then
  brew install protobuf jq
fi
