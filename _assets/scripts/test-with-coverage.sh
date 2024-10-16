#!/usr/bin/env bash
set -eu

packages=""
coverage_file_path="$(mktemp coverage.out.rerun.XXXXXXXXXX --tmpdir="${TEST_WITH_COVERAGE_REPORTS_DIR}")"
count=1

# This is a hack to workaround gotestsum behaviour. When using a --raw-command,
# gotestsum will only pass the package when rerunning a test. Otherwise we should pass the package ourselves.
# https://github.com/gotestyourself/gotestsum/blob/03568ab6d48faabdb632013632ac42687b5f17d1/cmd/main.go#L331-L336
if [[ "$*" != *"-test.run"* ]]; then
  packages="${TEST_WITH_COVERAGE_PACKAGES}"
  count=${TEST_WITH_COVERAGE_COUNT}
fi

go test -json \
  ${packages} \
  -count=${count} \
  -covermode=atomic \
  -coverprofile="${coverage_file_path}" \
  -coverpkg ./... \
  "$@"
