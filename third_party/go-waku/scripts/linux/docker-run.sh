#!/bin/bash

docker run -it --rm \
  --cap-add SYS_ADMIN \
  --security-opt apparmor:unconfined \
  --device /dev/fuse \
  -u  jenkins:$(id -g) \
  -v "${PWD}:/go-waku" \
  -w /go-waku \
  wakuorg/gowaku-linux-pkgs:latest \
  /go-waku/scripts/linux/build-pkgs.sh
