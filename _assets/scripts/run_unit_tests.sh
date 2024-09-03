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
  local single_timeout="$4"
  local log_message="$5"

  local output_file="test_${iteration}.log"
  local coverage_file="test_${iteration}.coverage.out"
  local report_file="report_${iteration}.xml"
  local rerun_report_file="report_rerun_fails_${iteration}.txt"
  local exit_code_file="exit_code_${iteration}.txt"
  local timeout="$(( single_timeout * count))m"

  echo -e "${GRN}Testing:${RST} ${log_message}. Iteration ${iteration}. -test.count=${count}. Timeout: ${timeout}"

  gotestsum_flags="${GOTESTSUM_EXTRAFLAGS}"
  if [[ "${CI}" == 'true' ]]; then
    gotestsum_flags="${gotestsum_flags} --junitfile=${report_file} --rerun-fails-report=${rerun_report_file}"
  fi

  # Prepare env variables for `test-with-coverage.sh`
  export TEST_WITH_COVERAGE_PACKAGES="${packages}"
  export TEST_WITH_COVERAGE_COUNT="${count}"
  export TEST_WITH_COVERAGE_REPORTS_DIR="$(mktemp -d)"

  # Cleanup previous coverage reports
  rm -f "${TEST_WITH_COVERAGE_REPORTS_DIR}/coverage.out.rerun.*"

  # Run tests
  gotestsum --packages="${packages}" ${gotestsum_flags} --raw-command -- \
    ./_assets/scripts/test-with-coverage.sh \
    ${GOTEST_EXTRAFLAGS} \
    -timeout "${timeout}" \
    -tags "${BUILD_TAGS}" | \
    redirect_stdout "${output_file}"

  local go_test_exit=$?

  # Merge package coverage results
  go run ./cmd/test-coverage-utils/gocovmerge.go ${TEST_WITH_COVERAGE_REPORTS_DIR}/coverage.out.rerun.* > ${coverage_file}
  rm -f "${COVERAGE_REPORTS_DIR}/coverage.out.rerun.*"

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

DEFAULT_TIMEOUT_MINUTES=5
PROTOCOL_TIMEOUT_MINUTES=45
if [[ "${UNIT_TEST_PACKAGES}" == *"github.com/status-im/status-go/protocol"* ]]; then
  HAS_PROTOCOL_PACKAGE=true
fi

if [[ $HAS_PROTOCOL_PACKAGE != 'true' ]]; then
  # This is the default single-line flow for testing all packages
  # The `else` branch is temporary and will be removed once the `protocol` package runtime is optimized.
  run_test_for_packages "${UNIT_TEST_PACKAGES}" "0" "${UNIT_TEST_COUNT}" "${DEFAULT_TIMEOUT_MINUTES}" "All packages"
else
  # Spawn a process to test all packages except `protocol`
  UNIT_TEST_PACKAGES=$(echo "${UNIT_TEST_PACKAGES}" | grep -v '.*/protocol$$')
  run_test_for_packages "${UNIT_TEST_PACKAGES}" "0" "${UNIT_TEST_COUNT}" "${DEFAULT_TIMEOUT_MINUTES}" "All packages except 'protocol'" &

  # Spawn separate processes to run `protocol` package
  for ((i=1; i<=UNIT_TEST_COUNT; i++)); do
    run_test_for_packages github.com/status-im/status-go/protocol "${i}" 1 "${PROTOCOL_TIMEOUT_MINUTES}" "Only 'protocol' package" &
  done

  wait
fi

# Gather test coverage results
merged_coverage_report="coverage_merged.out"
final_coverage_report="c.out" # Name expected by cc-test-reporter
coverage_reports=$(find . -iname "*.coverage.out")
rm -f ${final_coverage_report} ${merged_coverage_report}

echo -e "${GRN}Gathering test coverage results: ${RST} output: ${merged_coverage_report}, input: ${coverage_reports}"
echo $coverage_reports | xargs go run ./cmd/test-coverage-utils/gocovmerge.go > ${merged_coverage_report}

# Filter out test coverage for packages in ./cmd
echo -e "${GRN}Filtering test coverage packages:${RST} ./cmd"
grep -v '^github.com/status-im/status-go/cmd/' ${merged_coverage_report} > ${final_coverage_report}

# Generate HTML coverage report
echo -e "${GRN}Generating HTML coverage report${RST}"
go tool cover -html ${final_coverage_report} -o test-coverage.html

# Upload coverage report to CodeClimate
if [[ $UNIT_TEST_REPORT_CODECLIMATE == 'true' ]]; then
  echo -e "${GRN}Uploading coverage report to CodeClimate${RST}"
  # https://docs.codeclimate.com/docs/jenkins#jenkins-ci-builds
  GIT_COMMIT=$(git log | grep -m1 -oE '[^ ]+$')
  cc-test-reporter format-coverage --prefix=github.com/status-im/status-go # To generate 'coverage/codeclimate.json'
  cc-test-reporter after-build --prefix=github.com/status-im/status-go
fi

# Generate report with test stats
shopt -s globstar nullglob # Enable recursive globbing
if [[ "${UNIT_TEST_COUNT}" -gt 1 ]]; then
  for exit_code_file in "${GIT_ROOT}"/**/exit_code_*.txt; do
    read exit_code < "${exit_code_file}"
    if [[ "${exit_code}" -ne 0 ]]; then
      echo -e "${GRN}Generating test stats${RST}, exit code: ${exit_code}"
      mkdir -p "${GIT_ROOT}/reports"
      "${GIT_ROOT}/_assets/scripts/test_stats.py" | tee "${GIT_ROOT}/reports/test_stats.txt"
      exit ${exit_code}
    fi
  done
fi

echo -e "${GRN}Testing finished${RST}"
