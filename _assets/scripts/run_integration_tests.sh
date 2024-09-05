#!/usr/bin/env bash

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)
source "${GIT_ROOT}/_assets/scripts/colors.sh"
source "${GIT_ROOT}/_assets/scripts/codecov.sh"

echo -e "${GRN}Running integration tests${RST}"

root_path="${GIT_ROOT}/integration-tests"
coverage_reports_path="${root_path}/coverage"
test_results_path="${root_path}/reports"
log_file="${root_path}/tests.log"

echo -e "${GRN}root_path:${RST} ${root_path}"
echo -e "${GRN}coverage_reports_path:${RST} ${coverage_reports_path}"
echo -e "${GRN}test_results_path:${RST} ${test_results_path}"
echo -e "${GRN}log_file:${RST} ${log_file}"

# Create directories
mkdir -p "${GIT_ROOT}/integration-tests/coverage"

# Cleanup any previous coverage reports
rm -rf "${coverage_reports_path}"
rm -rf "${test_results_path}"

# Run integration tests
echo -e "${GRN}Running tests${RST}, HEAD: $(git rev-parse HEAD)"
docker-compose \
  -f ${root_path}/docker-compose.anvil.yml \
  -f ${root_path}/docker-compose.test.status-go.yml \
  up -d --build --remove-orphans > ${log_file}

# Save logs
echo -e "${GRN}Saving logs${RST}"
docker-compose \
  -f ${root_path}/docker-compose.anvil.yml \
  -f ${root_path}/docker-compose.test.status-go.yml \
  logs -f tests-rpc > ${log_file}

# Retrieve exit code
exit_code=$(docker inspect integration-tests_tests-rpc_1 -f '{{.State.ExitCode}}');

# Stop and remove containers
echo -e "${GRN}Stopping docker containers${RST}"
docker-compose \
  -f ${root_path}/docker-compose.anvil.yml \
  -f ${root_path}/docker-compose.test.status-go.yml \
  down > ${log_file}

# Early exit if tests failed
if [[ "$exit_code" -ne 0 ]]; then
  exit $exit_code
fi

# Prepare coverage reports
binary_coverage_reports_path="${coverage_reports_path}/binary"
merged_coverage_reports_path="${coverage_reports_path}/merged"
full_coverage_profile="${coverage_reports_path}/coverage.out"

# Clean merged reports directory
mkdir -p "${merged_coverage_reports_path}"

# Merge coverage reports
go tool covdata merge -i="${binary_coverage_reports_path}" -o="${merged_coverage_reports_path}"

# Convert coverage reports to profile
go tool covdata textfmt -i="${merged_coverage_reports_path}" -o="${full_coverage_profile}"

# Upload reports to Codecov
if [[ ${INTEGRATION_TESTS_REPORT_CODECOV} == 'true' ]]; then
# Docs: https://go.dev/blog/integration-test-coverage
  report_to_codecov "${test_results_path}/*.xml" "${full_coverage_profile}" "integration"
fi
