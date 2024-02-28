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

UNIT_TEST_PACKAGE_TIMEOUT="2m"
UNIT_TEST_PACKAGE_TIMEOUT_EXTENDED="30m"

redirect_stdout() {
  output_file=$1

  if [[ "${CI}" == 'true' ]]; then
    cat > "${output_file}";
  else
    tee "${output_file}";
  fi
}

has_extended_timeout() {
  local package
  for package in ${UNIT_TEST_PACKAGES_WITH_EXTENDED_TIMEOUT}; do
    if [[ "$1" == "${package}" ]]; then
      return 0
    fi
  done
  return 1
}

run_test_for_package() {
  local package=$1
  local iteration=$2
  echo -e "${GRN}Testing:${RST} ${package} Iteration:${iteration}"
  package_dir=$(go list -f "{{.Dir}}" "${package}")
  output_file="${package_dir}/test_${iteration}.log"

  if has_extended_timeout "${package}"; then
    package_timeout="${UNIT_TEST_PACKAGE_TIMEOUT_EXTENDED}"
  else
    package_timeout="${UNIT_TEST_PACKAGE_TIMEOUT}"
  fi

  local report_file="${package_dir}/report_${iteration}.xml"
  local rerun_report_file="${package_dir}/report_rerun_fails_${iteration}.txt"

  gotestsum_flags="${GOTESTSUM_EXTRAFLAGS}"
  if [[ "${CI}" == 'true' ]]; then
    gotestsum_flags="${gotestsum_flags} --junitfile=${report_file} --rerun-fails-report=${rerun_report_file}"
  fi

  gotestsum --packages="${package}" ${gotestsum_flags} -- \
    -v ${GOTEST_EXTRAFLAGS} \
    -timeout "${package_timeout}" \
    -count 1 \
    -tags "${BUILD_TAGS}" | \
    redirect_stdout "${output_file}"
  return $?
}

last_failing_exit_code=0

for package in ${UNIT_TEST_PACKAGES}; do
  for ((i=1; i<=UNIT_TEST_COUNT; i++)); do
    run_test_for_package "${package}" "${i}"
    go_test_exit=$?

    if [[ "${go_test_exit}" -ne 0 ]]; then
      if [[ "${CI}" == 'true' ]]; then
        echo -e "${YLW}Failed, see the log:${RST} ${BLD}${output_file}${RST}"
      fi

      if [[ "$UNIT_TEST_FAILFAST" == 'true' ]]; then
        exit "${go_test_exit}"
      fi

      last_failing_exit_code="${go_test_exit}"
    fi
  done
done

if [[ "${last_failing_exit_code}" -ne 0 ]]; then
  if [[ "${UNIT_TEST_COUNT}" -gt 1 ]]; then
    mkdir -p "${GIT_ROOT}/reports"
    "${GIT_ROOT}/_assets/scripts/test_stats.py" | redirect_stdout "${GIT_ROOT}/reports/test_stats.txt"
  fi

  exit "${last_failing_exit_code}"
fi
