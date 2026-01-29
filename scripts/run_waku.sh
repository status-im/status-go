#!/usr/bin/env bash

set -euo pipefail

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)
source "${GIT_ROOT}/scripts/colors.sh"
source "${GIT_ROOT}/scripts/codecov.sh"

root_path="${GIT_ROOT}/tests-functional"

identifier=${BUILD_ID:-$(git rev-parse --short HEAD)}
container_name="waku-${identifier}"

echo -e "${YLW}Starting waku node...${RST}"

IP_ADDRESS=$(ip -o -4 addr show up primary scope global | awk '{print $4}' | cut -d/ -f1 | head -n1);
docker run -d --name ${container_name} \
  -p 60000:60000/tcp -p 9000:9000/udp -p 8645:8645/tcp \
  harbor.status.im/wakuorg/nwaku:v0.36.0 \
  --tcp-port=60000 --discv5-discovery=true \
  --cluster-id=16 --shard=32 --shard=64 \
  --nat=extip:${IP_ADDRESS} --discv5-udp-port=9000 \
  --rest-address=0.0.0.0 --store

echo -e "${GRN}Waku node started.${RST}"

read -p "Press any button to exit..." -n 1 -r
echo

cleanup() {
  echo -e "${YLW}Removing containers...${RST}"
  docker ps -a --filter "name=${container_name}" -q | xargs -r docker rm -f
  echo -e "${GRN}DONE!${RST}"
}

trap cleanup EXIT
