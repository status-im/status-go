#!/usr/bin/env bash
set -o pipefail

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)

source "${GIT_ROOT}/_assets/scripts/colors.sh"

for package in ${UNIT_TEST_PACKAGES}; do
  echo -e "${GRN}Testing:${RST} ${package}"
  package_dir=$(go list -f "{{.Dir}}" "${package}")
  output_file=${package_dir}/test.log

  go test -tags "${BUILD_TAGS}" -timeout 30m -v -failfast "${package}" ${GOTEST_EXTRAFLAGS} | \
     if [ "${CI}" = "true" ]; then cat > "${output_file}"; else tee "${output_file}"; fi
  go_test_exit=$?

  if [ "${CI}" = "true" ]; then
    go-junit-report -in "${output_file}" -out "${package_dir}"/report.xml
  fi

  if [ ${go_test_exit} -ne 0 ]; then
    echo -e "${YLW}Failed, see the log:${RST} ${BLD}${output_file}${RST}"
    exit "${go_test_exit}"
  fi
done
