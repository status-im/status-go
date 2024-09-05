#!/usr/bin/env bash

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)
source "${GIT_ROOT}/_assets/scripts/colors.sh"
source "${GIT_ROOT}/_assets/scripts/codecov.sh"

root_path=./integration-tests
coverage_reports_path="${root_path}/coverage"

# Cleanup any previous coverage reports
rm -rf ${coverage_reports_path}

# Run integration tests
echo -e "${GRN}Running integration tests${RST}, HEAD: $(git rev-parse HEAD)"
docker-compose \
  -f ${root_path}/docker-compose.anvil.yml \
  -f ${root_path}/docker-compose.test.status-go.yml \
  up -d --build --remove-orphans;

# Save logs
echo -e "${GRN}Saving logs${RST}"
docker-compose \
  -f ${root_path}/docker-compose.anvil.yml \
  -f ${root_path}/docker-compose.test.status-go.yml \
  logs -f tests-rpc;

# Retrieve exit code
exit_code=$(docker inspect integration-tests_tests-rpc_1 -f '{{.State.ExitCode}}');

# Stop and remove containers
echo -e "${GRN}Stopping docker containers${RST}"
docker-compose \
  -f ${root_path}/docker-compose.anvil.yml \
  -f ${root_path}/docker-compose.test.status-go.yml \
  down;

# Early exit if tests failed
if [[ "$exit_code" -ne 0 ]]; then
  exit $exit_code
fi

# Report to Codecov
if [[ ${INTEGRATION_TESTS_REPORT_CODECOV} == 'true' ]]; then
  # Docs: https://go.dev/blog/integration-test-coverage
  binary_coverage_reports_path="${coverage_reports_path}/binary"
  merged_coverage_reports_path="${coverage_reports_path}/merged"
  full_coverage_profile="${coverage_reports_path}/coverage.out"
  test_results_path="${root_path}/reports"

  # Clean merged reports directory
  mkdir -p ${merged_coverage_reports_path}

  # Merge coverage reports
  go tool covdata merge -i=${binary_coverage_reports_path} -o=${merged_coverage_reports_path}

  # Convert coverage reports to profile
  go tool covdata textfmt -i=${merged_coverage_reports_path} -o=${full_coverage_profile}

  # Upload reports to Codecov
  report_to_codecov "${test_results_path}/*.xml" "${full_coverage_profile}" "integration"
fi
