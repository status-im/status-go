#!/usr/bin/env bash
set -o pipefail

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)

source "${GIT_ROOT}/_assets/scripts/colors.sh"

if [[ $UNIT_TEST_RERUN_FAILS == 'true' ]]; then
  GOTESTSUM_EXTRAFLAGS="${GOTESTSUM_EXTRAFLAGS} --rerun-fails"
elif [[ $UNIT_TEST_FAILFAST == 'true' ]]; then
  GOTEST_EXTRAFLAGS="${GOTEST_EXTRAFLAGS} -failfast"
fi

if [[ $UNIT_TEST_USE_DEVELOPMENT_LOGGER == 'false' ]]; then
  if [[ -z $BUILD_TAGS ]]; then
    BUILD_TAGS="test_silent"
  else
    BUILD_TAGS="${BUILD_TAGS},test_silent"
  fi
fi

if [[ -z "${UNIT_TEST_COUNT}" ]]; then
  UNIT_TEST_COUNT=1
fi

redirect_stdout() {
  output_file=$1

  if [[ "${CI}" == 'true' ]]; then
    cat > "${output_file}";
  else
    tee "${output_file}";
  fi
}

run_test_for_packages() {
  local packages="$1"
  local iteration="$2"
  local count="$3"
  local log_message="$4"

  local output_file="test_${iteration}.log"
  local coverage_file="test_${iteration}.coverage.out"
  local report_file="report_${iteration}.xml"
  local rerun_report_file="report_rerun_fails_${iteration}.txt"
  local exit_code_file="exit_code_${iteration}.txt"

  echo -e "${GRN}Testing:${RST} ${log_message}. Iteration ${iteration}. -test.count=${count}"

  gotestsum_flags="${GOTESTSUM_EXTRAFLAGS}"
  if [[ "${CI}" == 'true' ]]; then
    gotestsum_flags="${gotestsum_flags} --junitfile=${report_file} --rerun-fails-report=${rerun_report_file}"
  fi

  # Cleanup previous coverage reports
  rm -f coverage.out.rerun.*

  # Prepare env variables for `test-with-coverage.sh`
  export TEST_WITH_COVERAGE_PACKAGES="${packages}"
  export TEST_WITH_COVERAGE_COUNT="${count}"

  # Run tests
  gotestsum --packages="${packages}" ${gotestsum_flags} --raw-command -- \
    ./_assets/scripts/test-with-coverage.sh \
    ${GOTEST_EXTRAFLAGS} \
    -timeout 40m \
    -tags "${BUILD_TAGS}" | \
    redirect_stdout "${output_file}"

  local go_test_exit=$?

  # Merge package coverage results
  go run ./cmd/test-coverage-utils/gocovmerge.go coverage.out.rerun.* > ${coverage_file}

  # Cleanup coverage reports
  rm -f coverage.out.rerun.*

  echo "${go_test_exit}" > "${exit_code_file}"
  if [[ "${go_test_exit}" -ne 0 ]]; then
    if [[ "${CI}" == 'true' ]]; then
      echo -e "${YLW}Failed, see the log:${RST} ${BLD}${output_file}${RST}"
    fi
  fi

  return ${go_test_exit}
}

if [[ $UNIT_TEST_REPORT_CODECLIMATE == 'true' ]]; then
	cc-test-reporter before-build
fi

rm -rf ./**/*.coverage.out

echo -e "${GRN}Testing HEAD:${RST} $(git rev-parse HEAD)"

run_test_for_packages "${UNIT_TEST_PACKAGES}" "${UNIT_TEST_COUNT}" "0" "All packages except 'protocol'" &

# NOTE: Run `protocol` package manually, because it lasts for 30 minutes.
# Running -test.count=20 for it takes too much time, so we spawn separate processes for it.
# This can be removed when the runtime of `protocol` package is optimized.
for ((i=1; i<=UNIT_TEST_COUNT; i++)); do
  run_test_for_packages github.com/status-im/status-go/protocol 1 "${i}" "Only 'protocol' package" &
done

wait

# Gather test coverage results
rm -f c.out c-full.out
go run ./cmd/test-coverage-utils/gocovmerge.go $(find -iname "*.coverage.out") >> c-full.out

# Filter out test coverage for packages in ./cmd
grep -v '^github.com/status-im/status-go/cmd/' c-full.out > c.out

# Generate HTML coverage report
go tool cover -html c.out -o test-coverage.html

if [[ $UNIT_TEST_REPORT_CODECLIMATE == 'true' ]]; then
  # https://docs.codeclimate.com/docs/jenkins#jenkins-ci-builds
  GIT_COMMIT=$(git log | grep -m1 -oE '[^ ]+$')
  cc-test-reporter format-coverage --prefix=github.com/status-im/status-go # To generate 'coverage/codeclimate.json'
  cc-test-reporter after-build --prefix=github.com/status-im/status-go
fi

shopt -s globstar nullglob # Enable recursive globbing
if [[ "${UNIT_TEST_COUNT}" -gt 1 ]]; then
  for exit_code_file in "${GIT_ROOT}"/**/exit_code_*.txt; do
    read exit_code < "${exit_code_file}"
    if [[ "${exit_code}" -ne 0 ]]; then
      mkdir -p "${GIT_ROOT}/reports"
      "${GIT_ROOT}/_assets/scripts/test_stats.py" | redirect_stdout "${GIT_ROOT}/reports/test_stats.txt"
      exit ${exit_code}
    fi
  done
fi
