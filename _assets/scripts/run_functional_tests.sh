#!/usr/bin/env bash

set -o nounset

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)
source "${GIT_ROOT}/_assets/scripts/colors.sh"
source "${GIT_ROOT}/_assets/scripts/codecov.sh"

echo -e "${GRN}Running functional tests${RST}"

root_path="${GIT_ROOT}/tests-functional"
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
timestamp=$(date +%s)

# Run functional tests
echo -e "${GRN}Running tests${RST}, HEAD: $(git rev-parse HEAD)"
docker-compose -p ${timestamp} ${all_compose_files} up -d --build --remove-orphans

echo -e "${GRN}Running tests-rpc${RST}" # Follow the logs, wait for them to finish
docker-compose -p ${timestamp} ${all_compose_files} logs -f tests-rpc > "${root_path}/tests-rpc.log"

# Stop containers
echo -e "${GRN}Stopping docker containers${RST}"
docker-compose -p ${timestamp} ${all_compose_files} stop

# Save logs
echo -e "${GRN}Saving logs${RST}"
docker-compose -p ${timestamp} ${all_compose_files} logs status-go > "${root_path}/statusd.log"
docker-compose -p ${timestamp} ${all_compose_files} logs status-backend > "${root_path}/status-backend.log"

if [ "$(uname)" = "Darwin" ]; then
    separator="-"
else
    separator="_"
fi

# Retrieve exit code
exit_code=$(docker inspect ${timestamp}${separator}tests-rpc${separator}1 -f '{{.State.ExitCode}}');

# Cleanup containers
echo -e "${GRN}Removing docker containers${RST}"
docker-compose -p ${timestamp} ${all_compose_files} down

# Collect coverage reports
echo -e "${GRN}Collecting code coverage reports${RST}"
full_coverage_profile="${coverage_reports_path}/coverage.out"
go tool covdata merge -i="${binary_coverage_reports_path}" -o="${merged_coverage_reports_path}"
go tool covdata textfmt -i="${merged_coverage_reports_path}" -o="${full_coverage_profile}"
convert_coverage_to_html "${full_coverage_profile}" "${coverage_reports_path}/coverage.html"

# Upload reports to Codecov
if [[ ${FUNCTIONAL_TESTS_REPORT_CODECOV} == 'true' ]]; then
  report_to_codecov "${test_results_path}/*.xml" "${full_coverage_profile}" "functional"
fi

echo -e "${GRN}Testing finished${RST}"
exit $exit_code