#!/usr/bin/env bash

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)
source "${GIT_ROOT}/_assets/scripts/colors.sh"

echo -e "${GRN}Removing mockgen files from ./mock directories${RST}" # excluding ./vendor and ./contracts directories
find . \
  \( \
    -type d -name "mock" \
    -and -not -path "./vendor/*" \
    -and -not -path "./contracts/*" \
  \) \
  -exec rm -rf {} +

echo -e "${GRN}Removing mock.go files${RST}" # In theory this is only ./transactions/fake/mock.go
find . \
  -name "mock.go" \
  -and -not -path "./vendor/*" \
  -exec rm -f {} +

echo -e "${GRN}Removing protoc and go-bindata files${RST}"
find . \
  \( \
    -name '*.pb.go' \
    -or -name 'bindata.go' \
    -or -name 'migrations.go' \
    -or -name 'messenger_handlers.go' \
  \) \
  -and -not -path './vendor/*' \
  -exec rm -f {} +