#!/bin/bash

if [ -x "$(command -v apt)" ]; then
  apt install -y protobuf-compiler
fi

if [ -x "$(command -v pacman)" ]; then
  pacman -Sy protobuf --noconfirm
fi

if [ -x "$(command -v brew)" ]; then
  brew install protobuf
fi
