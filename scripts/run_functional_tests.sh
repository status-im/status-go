#!/usr/bin/env bash

set -o nounset

unset LD_LIBRARY_PATH

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)
source "${GIT_ROOT}/scripts/colors.sh"
source "${GIT_ROOT}/scripts/codecov.sh"

: "${FUNCTIONAL_TESTS_LOG_LEVEL:=INFO}"
: "${FUNCTIONAL_TESTS_REPORT_CODECOV:=false}"
: "${FUNCTIONAL_TESTS_BUILD_TAGS:=gowaku_no_rln}"
: "${FUNCTIONAL_TESTS_USE_LOGOS_STORAGE:=false}"

echo -e "${GRN}Running functional tests${RST}"

root_path="${GIT_ROOT}/tests-functional"
coverage_reports_path="${root_path}/coverage"
binary_coverage_reports_path="${coverage_reports_path}/binary"
merged_coverage_reports_path="${coverage_reports_path}/merged"
test_results_path="${root_path}/reports"
logs_path="${root_path}/logs"

# Cleanup any previous coverage reports
rm -rf "${coverage_reports_path}"
rm -rf "${test_results_path}"
rm -rf "${logs_path}"

# Create directories
mkdir -p "${binary_coverage_reports_path}"
mkdir -p "${merged_coverage_reports_path}"
mkdir -p "${test_results_path}"
mkdir -p "${logs_path}"

all_compose_files="-f ${root_path}/docker-compose.anvil.yml -f ${root_path}/docker-compose.waku.yml"
identifier=${FUNCTIONAL_TESTS_CONTAINER_PREFIX:-"status-go-func-tests-$(git rev-parse --short HEAD)"}
project_name="${identifier,,}"
image_name="statusgo-${identifier,,}"

# Remove orphans
echo -e "${GRN}Cleanup old containers${RST}"
docker ps -a --filter "name=${project_name}" --filter "status=exited" -q | xargs -r docker rm -f

# Build statusgo image
echo -e "${GRN}Building status-go${RST}"
build_tags="${FUNCTIONAL_TESTS_BUILD_TAGS}"
pytest_marker_expr="rpc and not logos_storage"
if [[ "${FUNCTIONAL_TESTS_USE_LOGOS_STORAGE}" == "true" ]]; then
  build_tags="${build_tags} use_logos_storage"
  pytest_marker_expr="rpc"
  if [[ -n "${IN_NIX_SHELL:-}" && -n "${LIBSTORAGE_PATH:-}" ]]; then
    mkdir -p "${GIT_ROOT}/libs"
    if [[ -f "${LIBSTORAGE_PATH}/lib/libstorage.so" ]]; then
      cp "${LIBSTORAGE_PATH}/lib/libstorage.so" "${GIT_ROOT}/libs/libstorage.so"
      if [[ -f "${LIBSTORAGE_PATH}/include/libstorage.h" ]]; then
        cp "${LIBSTORAGE_PATH}/include/libstorage.h" "${GIT_ROOT}/libs/libstorage.h"
        echo -e "${GRN}Prepared ./libs/libstorage.so and ./libs/libstorage.h from \$LIBSTORAGE_PATH${RST}"
      else
        echo -e "${YEL}No libstorage.h at ${LIBSTORAGE_PATH}/include; Docker build may need make fetch-storage to download headers.${RST}"
      fi
    else
      echo -e "${YEL}No libstorage.so at ${LIBSTORAGE_PATH}/lib; Docker build will rely on make fetch-storage.${RST}"
    fi
  fi
fi
docker build . \
  --build-arg "build_flags=-cover" \
  --build-arg "build_tags=${build_tags}" \
  --build-arg "use_logos_storage=${FUNCTIONAL_TESTS_USE_LOGOS_STORAGE}" \
  --build-arg "enable_go_cache=false" \
  --tag "${image_name}"

if [[ $? -ne 0 ]]; then
    echo -e "${RED}Docker build failed. Exiting.${RST}"
    exit 1
fi

# Run docker
echo -e "${GRN}Running status-go external dependencies${RST}"
docker compose -p ${project_name} ${all_compose_files} up -d --build --remove-orphans

# Wait for wakufleet-scanner to finish before running tests. If it fails,
# there's no point in starting tests because wakufleetconfig.json will not
# be available for status-backend.
echo -e "${GRN}Waiting for wakufleet-scanner to complete${RST}"
scanner_container_id=$(docker compose -p ${project_name} ${all_compose_files} ps -q wakufleet-scanner || true)

if [[ -z "${scanner_container_id}" ]]; then
  echo -e "${RED}wakufleet-scanner service container not found. Make sure docker-compose.waku.yml defines it correctly.${RST}"
  exit 1
fi

scanner_exit_code=$(docker wait "${scanner_container_id}")

if [[ "${scanner_exit_code}" -ne 0 ]]; then
  echo -e "${RED}wakufleet-scanner failed with exit code ${scanner_exit_code}. See logs below:${RST}"
  docker logs "${scanner_container_id}" || true
  echo -e "${RED}Aborting functional tests because wakufleetconfig.json was not generated successfully.${RST}"
  exit 1
fi

echo -e "${GRN}wakufleet-scanner completed successfully${RST}"

# Set up virtual environment
venv_path="${root_path}/.venv"

if [[ -d "${venv_path}" ]]; then
    echo -e "${GRN}Using existing virtual environment${RST}"
else
    echo -e "${GRN}Creating new virtual environment${RST}"
    python3 -m venv "${venv_path}"
fi

source "${venv_path}/bin/activate"

# Upgrade pip and install requirements
echo -e "${GRN}Installing dependencies${RST}"
pip install --upgrade pip
pip install -r "${root_path}/requirements.txt"

# Run functional tests
echo -e "${GRN}Running tests${RST}, HEAD: $(git rev-parse HEAD)"
pytest --reruns 2 -m "${pytest_marker_expr}" -c "${root_path}/pytest.ini" -n 12 \
  --dist loadgroup\
  --log-cli-level="${FUNCTIONAL_TESTS_LOG_LEVEL}" \
  --docker_project_name="${project_name}" \
  --docker-image=${image_name} \
  --codecov_dir="${binary_coverage_reports_path}" \
  --logs-dir="${logs_path}" \
  --junitxml="${test_results_path}/report.xml" \
  ${root_path}
exit_code=$?

# Stop containers
echo -e "${GRN}Stopping docker containers${RST}"
docker compose -p ${project_name} ${all_compose_files} stop

# Cleanup containers
echo -e "${GRN}Removing docker containers${RST}"
docker compose -p ${project_name} ${all_compose_files} down

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
