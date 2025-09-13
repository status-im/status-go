#!/usr/bin/env bash

set -euo pipefail

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)
source "${GIT_ROOT}/_assets/scripts/colors.sh"
source "${GIT_ROOT}/_assets/scripts/codecov.sh"

: "${FUNCTIONAL_TESTS_LOG_LEVEL:=INFO}"

root_path="${GIT_ROOT}/tests-functional"
logs_path="${root_path}/logs"

all_compose_files="-f ${root_path}/docker-compose.anvil.yml -f ${root_path}/docker-compose.waku.yml"
identifier=${BUILD_ID:-$(git rev-parse --short HEAD)}
project_name="status-go-func-tests-${identifier}"
image_name="statusgo-${identifier}"

source "${GIT_ROOT}/_assets/scripts/functional_tests_commons.sh"

test_pattern="${1-}"

set_pyenv

list_tests_and_confirm "${test_pattern}"

remove_old_logs

remove_old_containers

start_services

wait_for_services 60

run_tests "${test_pattern}"

# Cleanup will be handled automatically by the trap
echo -e "${GRN}Testing finished${RST}"
exit $exit_code
