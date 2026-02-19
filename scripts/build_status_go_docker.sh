#!/usr/bin/env bash

# Use this script to build the status-go docker image that is slightly lighter
# than the one built by the default run_functional_tests.sh script and use it
# together with run_functional_tests_dev.sh, which makes running functional
# tests slightly more convenient locally during development.
set -o nounset

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)
source "${GIT_ROOT}/scripts/colors.sh"
source "${GIT_ROOT}/scripts/codecov.sh"

echo -e "${GRN}Building status-go docker image...${RST}"

identifier=${BUILD_TAG:-"status-go-func-tests-$(git rev-parse --short HEAD)"}
image_name="statusgo-${identifier,,}"

# Build statusgo image
echo -e "${GRN}Building status-go${RST}"
docker build . \
  --build-arg "build_tags='gowaku_no_rln'" \
  --build-arg "enable_go_cache=false" \
  --tag "${image_name}"

echo -e "${GRN}Building status-go docker image DONE!${RST}"
