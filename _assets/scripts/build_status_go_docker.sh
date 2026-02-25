#!/usr/bin/env bash

# Use this script to build the status-go docker image that is slightly lighter
# than the one built by the default run_functional_tests.sh script and use it
# together with run_functional_tests_dev.sh, which makes running functional
# tests slightly more convenient locally during development.
set -euo pipefail

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)
source "${GIT_ROOT}/_assets/scripts/colors.sh"
source "${GIT_ROOT}/_assets/scripts/codecov.sh"

: "${FUNCTIONAL_TESTS_BUILD_TAGS:=gowaku_no_rln}"
: "${FUNCTIONAL_TESTS_USE_LOGOS_STORAGE:=false}"

echo -e "${GRN}Building status-go docker image...${RST}"

identifier=${FUNCTIONAL_TESTS_CONTAINER_PREFIX:-"status-go-func-tests-$(git rev-parse --short HEAD)"}
image_name="statusgo-${identifier,,}"

build_tags="${FUNCTIONAL_TESTS_BUILD_TAGS}"
if [[ "${FUNCTIONAL_TESTS_USE_LOGOS_STORAGE}" == "true" ]]; then
  build_tags="${build_tags} use_logos_storage"
  if [[ -n "${IN_NIX_SHELL:-}" && -n "${LIBSTORAGE_PATH:-}" ]]; then
    mkdir -p "${GIT_ROOT}/libs"
    if [[ -f "${LIBSTORAGE_PATH}/lib/libstorage.so" ]]; then
      cp "${LIBSTORAGE_PATH}/lib/libstorage.so" "${GIT_ROOT}/libs/libstorage.so"
      echo -e "${GRN}Prepared ./libs/libstorage.so from \$LIBSTORAGE_PATH${RST}"
    else
      echo -e "${YEL}No libstorage.so at ${LIBSTORAGE_PATH}/lib; Docker build will rely on make fetch-libstorage.${RST}"
    fi
  fi
fi

# Build statusgo image
echo -e "${GRN}Building status-go${RST}"
docker build . \
  --build-arg "build_tags=${build_tags}" \
  --build-arg "use_logos_storage=${FUNCTIONAL_TESTS_USE_LOGOS_STORAGE}" \
  --build-arg "enable_go_cache=false" \
  --tag "${image_name}"

echo -e "${GRN}Building status-go docker image DONE!${RST}"
