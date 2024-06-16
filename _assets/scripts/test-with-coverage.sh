#!/usr/bin/env bash
set -eu
coverage_file_path="${PACKAGE_DIR}/$(mktemp coverage.out.rerun.XXXXXXXXXX)"
go test -json \
  -covermode=atomic \
  -coverprofile="${coverage_file_path}" \
  "$@"
