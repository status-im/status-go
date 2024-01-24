#!/usr/bin/env bash
set -o pipefail

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)

source "${GIT_ROOT}/_assets/scripts/colors.sh"

if [[ $UNIT_TEST_FAILFAST == 'true' ]]; then
  GOTEST_EXTRAFLAGS="${GOTEST_EXTRAFLAGS} -failfast"
fi

if [[ -z "${UNIT_TEST_COUNT}" ]]; then
  UNIT_TEST_COUNT=1
fi

UNIT_TEST_PACKAGE_TIMEOUT="$((UNIT_TEST_COUNT * 2))m"
UNIT_TEST_PACKAGE_TIMEOUT_EXTENDED="$((UNIT_TEST_COUNT * 30))m"

redirect_stdout() {
  output_file=$1

  if [[ "${CI}" == 'true' ]];
  then
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

last_failing_exit_code=0

for package in ${UNIT_TEST_PACKAGES}; do
  echo -e "${GRN}Testing:${RST} ${package}"
  package_dir=$(go list -f "{{.Dir}}" "${package}")
  output_file=${package_dir}/test.log

  if has_extended_timeout "${package}"; then
    package_timeout="${UNIT_TEST_PACKAGE_TIMEOUT_EXTENDED}"
  else
    package_timeout="${UNIT_TEST_PACKAGE_TIMEOUT}"
  fi

  go test "${package}" -v ${GOTEST_EXTRAFLAGS} \
    -timeout "${package_timeout}" \
    -count "${UNIT_TEST_COUNT}" \
    -tags "${BUILD_TAGS}" | \
    redirect_stdout "${output_file}"
  go_test_exit=$?

  if [[ "${CI}" == 'true' ]]; then
    go-junit-report -in "${output_file}" -out "${package_dir}"/report.xml
  fi

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

if [[ "${last_failing_exit_code}" -ne 0 ]]; then
  if [[ "${UNIT_TEST_COUNT}" -gt 1 ]]; then
    mkdir -p "${GIT_ROOT}/reports"
    "${GIT_ROOT}/_assets/scripts/test_stats.py" | redirect_stdout "${GIT_ROOT}/reports/test_stats.txt"
  fi

  exit "${last_failing_exit_code}"
fi
