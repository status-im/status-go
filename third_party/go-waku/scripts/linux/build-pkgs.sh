#!/bin/bash

git config --global --add safe.directory /go-waku
make

cd /go-waku/scripts/linux

./fpm-build.sh
