#!/usr/bin/env bash

set -o nounset

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)
source "${GIT_ROOT}/_assets/scripts/colors.sh"
source "${GIT_ROOT}/_assets/scripts/codecov.sh"

echo -e "${GRN}Building status-go docker image...${RST}"

identifier=${BUILD_ID:-$(git rev-parse --short HEAD)}
image_name="statusgo-${identifier}"

# Build statusgo image
echo -e "${GRN}Building status-go${RST}"
docker build . \
  --build-arg "build_tags='gowaku_no_rln'" \
  --build-arg "enable_go_cache=false" \
  --tag "${image_name}"

echo -e "${GRN}Building status-go docker image DONE!${RST}"
