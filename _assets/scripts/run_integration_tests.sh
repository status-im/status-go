#!/usr/bin/env bash

set -x
set -o nounset

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)
source "${GIT_ROOT}/_assets/scripts/colors.sh"
source "${GIT_ROOT}/_assets/scripts/codecov.sh"

echo -e "${GRN}Running integration tests${RST}"

root_path="${GIT_ROOT}/integration-tests"
coverage_reports_path="${root_path}/coverage"
binary_coverage_reports_path="${coverage_reports_path}/binary"
merged_coverage_reports_path="${coverage_reports_path}/merged"
test_results_path="${root_path}/reports"

# Cleanup any previous coverage reports
rm -rf "${coverage_reports_path}"
rm -rf "${test_results_path}"

# Create directories
mkdir -p "${binary_coverage_reports_path}"
mkdir -p "${merged_coverage_reports_path}"
mkdir -p "${test_results_path}"

all_compose_files="-f ${root_path}/docker-compose.anvil.yml -f ${root_path}/docker-compose.test.status-go.yml"

# Run integration tests
echo -e "${GRN}Running tests${RST}, HEAD: $(git rev-parse HEAD)"
docker-compose ${all_compose_files} up -d --build --remove-orphans

# Save logs
echo -e "${GRN}Saving logs${RST}"
docker-compose ${all_compose_files} logs -f tests-rpc > "${root_path}/tests-rpc.log"
docker-compose ${all_compose_files} logs status-go > "${root_path}/statusd.log"
docker-compose ${all_compose_files} logs status-go-no-funds > "${root_path}/statusd-no-funds.log"

# Retrieve exit code
exit_code=$(docker inspect integration-tests_tests-rpc_1 -f '{{.State.ExitCode}}');

# Stop and remove containers
echo -e "${GRN}Stopping docker containers${RST}"
docker-compose ${all_compose_files} down

# Collect coverage reports
echo -e "${GRN}Collecting code coverage reports${RST}"
full_coverage_profile="${coverage_reports_path}/coverage.out"
go tool covdata merge -i="${binary_coverage_reports_path}" -o="${merged_coverage_reports_path}"
go tool covdata textfmt -i="${merged_coverage_reports_path}" -o="${full_coverage_profile}"
convert_coverage_to_html "${full_coverage_profile}" "${coverage_reports_path}/coverage.html"

# Upload reports to Codecov
if [[ ${INTEGRATION_TESTS_REPORT_CODECOV} == 'true' ]]; then
  report_to_codecov "${test_results_path}/*.xml" "${full_coverage_profile}" "integration"
fi

echo -e "${GRN}Testing finished${RST}"
exit $exit_code