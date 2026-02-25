#!/usr/bin/env bash

# Use this script to run the status-go functional tests in a local development environment.
# It is based on the default run_functional_tests.sh script but does not include
# building the status-go image, which can be done separately with build_status_go_docker.sh.
# This can make working with functional tests slightly more convenient during local development.
# The script also improves on the cleanup and provides options to run the tests in a more
# interactive manner. You can include the name of the test suite you want to run as an argument, for example:
# 
#   ./scripts/run_functional_tests_dev.sh TestInteractingWithChatMessages
#
# and the script will list all matching tests and will give you an option to cancel:
# 
#  ❯ ./scripts/run_functional_tests_dev.sh TestInteractingWithChatMessages
#    Using existing virtual environment
#    Installing dependencies
#    Discovering tests to be run...
#    Found 16 tests matching: TestInteractingWithChatMessages
#    Tests to execute:
#    1) test_pinned_messages[wakuV2LightClient_False]
#    2) test_pinned_messages[wakuV2LightClient_True]
#    3) test_pinned_messages_with_pagination[wakuV2LightClient_False]
#    4) test_pinned_messages_with_pagination[wakuV2LightClient_True]
#    5) test_edit_message[wakuV2LightClient_False]
#    6) test_edit_message[wakuV2LightClient_True]
#    7) test_delete_message[wakuV2LightClient_False]
#    8) test_delete_message[wakuV2LightClient_True]
#    9) test_delete_message_and_send[wakuV2LightClient_False]
#    10) test_delete_message_and_send[wakuV2LightClient_True]
#    11) test_delete_messages_by_chat_id[wakuV2LightClient_False]
#    12) test_delete_messages_by_chat_id[wakuV2LightClient_True]
#    13) test_delete_message_for_me_and_sync[wakuV2LightClient_False]
#    14) test_delete_message_for_me_and_sync[wakuV2LightClient_True]
#    15) test_update_message_outgoing_status[wakuV2LightClient_False]
#    16) test_update_message_outgoing_status[wakuV2LightClient_True]
#    Continue with execution? (y/n):
# 
# You can even run a single test directly, e.g:
# 
#  ❯ ./scripts/run_functional_tests_dev.sh test_pinned_messages_with_pagination[wakuV2LightClient_False]
#    Using existing virtual environment
#    Installing dependencies
#    Discovering tests to be run...
#    Found 1 tests matching: test_pinned_messages_with_pagination[wakuV2LightClient_False]
#    Tests to execute:
#    1) test_pinned_messages_with_pagination[wakuV2LightClient_False]
#    Continue with execution? (y/n):
# 

set -euo pipefail

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)
source "${GIT_ROOT}/_assets/scripts/colors.sh"
source "${GIT_ROOT}/_assets/scripts/codecov.sh"

: "${FUNCTIONAL_TESTS_LOG_LEVEL:=INFO}"
: "${FUNCTIONAL_TESTS_CONTAINER_PREFIX:=status-go-func-tests-$(git rev-parse --short HEAD)}"
: "${FUNCTIONAL_TESTS_BUILD_TAGS:=gowaku_no_rln}"
: "${FUNCTIONAL_TESTS_USE_LOGOS_STORAGE:=false}"

root_path="${GIT_ROOT}/tests-functional"
logs_path="${root_path}/logs"

all_compose_files="-f ${root_path}/docker-compose.anvil.yml -f ${root_path}/docker-compose.waku.yml"
identifier="${FUNCTIONAL_TESTS_CONTAINER_PREFIX}"
project_name="${identifier,,}"
image_name="statusgo-${identifier,,}"

source "${GIT_ROOT}/_assets/scripts/functional_tests_commons.sh"

test_pattern="${1-}"

set_pyenv

list_tests_and_confirm "${test_pattern}"

remove_old_logs

remove_old_containers

start_services

wait_for_waku_suite_scanner

wait_for_services 60

run_tests "${test_pattern}"

# Cleanup will be handled automatically by the trap
echo -e "${GRN}Testing finished${RST}"
exit $exit_code
