#!/bin/bash

if [ ! -z "$CI_SUDO_REQUIRED" ]; then
  sudo apt install -y protobuf-compiler libpcsclite-dev && exit
fi

if [ -x "$(command -v apt)" ]; then
  apt install -y protobuf-compiler libpcsclite-dev
fi

if [ -x "$(command -v pacman)" ]; then
  pacman -Sy protobuf --noconfirm
fi

if [ -x "$(command -v brew)" ]; then
  brew install protobuf
fi
