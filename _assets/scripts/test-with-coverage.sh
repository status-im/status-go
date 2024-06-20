#!/usr/bin/env bash
set -eu
coverage_file_path="$(mktemp coverage.out.rerun.XXXXXXXXXX --tmpdir="${PACKAGE_DIR}")"
go test -json \
  -covermode=atomic \
  -coverprofile="${coverage_file_path}" \
  -coverpkg ./... \
  "$@"
