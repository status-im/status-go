#!/usr/bin/env bash

set -o nounset

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)
source "${GIT_ROOT}/scripts/colors.sh"
source "${GIT_ROOT}/scripts/codecov.sh"

: "${FUNCTIONAL_TESTS_LOG_LEVEL:=INFO}"
: "${FUNCTIONAL_TESTS_REPORT_CODECOV:=false}"
: "${FUNCTIONAL_TESTS_BUILD_TAGS:=gowaku_no_rln}"
: "${USE_LOGOS_STORAGE:=false}"
: "${FUNCTIONAL_TESTS_MARKER:=rpc}"
: "${PEER_IMAGES:=}"
: "${PEER_REFS:=}"

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

dump_compose_logs_done=0
dump_compose_logs() {
  [[ ${dump_compose_logs_done} -eq 1 ]] && return 0
  dump_compose_logs_done=1
  {
    echo "=== docker compose ps ==="
    docker compose -p "${project_name}" ${all_compose_files} ps -a
    echo
    echo "=== docker compose logs ==="
    docker compose -p "${project_name}" ${all_compose_files} logs --no-color --timestamps
  } > "${logs_path}/compose.log" 2>&1 || true
}
trap dump_compose_logs EXIT

# Remove orphans
echo -e "${GRN}Cleanup old containers${RST}"
docker ps -a --filter "name=${project_name}" --filter "status=exited" -q | xargs -r docker rm -f

# Build statusgo image
echo -e "${GRN}Building status-go${RST}"
build_tags="${FUNCTIONAL_TESTS_BUILD_TAGS}"
if [[ "${USE_LOGOS_STORAGE}" == "true" ]]; then
  build_tags="${build_tags} use_logos_storage"
  if [[ -n "${LOGOS_STORAGE_LIB_DIR:-}" && ! -f "${LOGOS_STORAGE_LIB_DIR}/libstorage.so" ]]; then
    echo -e "${YEL}No libstorage.so at ${LOGOS_STORAGE_LIB_DIR}; build it first with make build-storage.${RST}"
  fi
fi
docker build . \
  --build-arg "build_flags=-cover" \
  --build-arg "build_tags=${build_tags}" \
  --build-arg "use_logos_storage=${USE_LOGOS_STORAGE}" \
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

# Resolve peer (old) backend images for cross-version compatibility tests.
peer_image_args=()
if [[ "${FUNCTIONAL_TESTS_MARKER}" == "compatibility" ]]; then
  echo -e "${GRN}Preparing peer images for compatibility tests${RST}"
  resolved_peer_images=()
  if [[ -n "${PEER_IMAGES}" ]]; then
    for img in ${PEER_IMAGES}; do
      docker pull "${img}" || { echo -e "${RED}Failed to pull peer image ${img}${RST}"; exit 1; }
      resolved_peer_images+=("${img}")
    done
  else
    if [[ -z "${PEER_REFS}" ]]; then
      echo -e "${RED}No peer refs available. Set PEER_REFS or PEER_IMAGES.${RST}"
      exit 1
    fi
    # TODO: drop worktree build once released backend images are published to Harbor.
    for ref in ${PEER_REFS}; do
      ref_slug="$(printf '%s' "${ref}" | tr -c 'A-Za-z0-9_.-' '-' | sed 's/^[-.]*//; s/[-.]*$//')"
      peer_image="statusgo-peer-${ref_slug}"
      if ! docker image inspect "${peer_image}" > /dev/null 2>&1; then
        echo -e "${GRN}Building peer backend ${ref} -> ${peer_image}${RST}"
        git rev-parse --verify --quiet "${ref}^{commit}" > /dev/null || git fetch --tags origin "${ref}"
        peer_worktree="$(mktemp -d)"
        git worktree add --force --detach "${peer_worktree}" "${ref}"
        if ! docker build "${peer_worktree}" --build-arg "build_tags=${FUNCTIONAL_TESTS_BUILD_TAGS}" --tag "${peer_image}"; then
          echo -e "${RED}Failed to build peer backend ${ref}${RST}"
          git worktree remove --force "${peer_worktree}" || true
          exit 1
        fi
        git worktree remove --force "${peer_worktree}"
      fi
      resolved_peer_images+=("${peer_image}")
    done
  fi
  for img in ${resolved_peer_images[@]+"${resolved_peer_images[@]}"}; do
    peer_image_args+=(--peer-docker-image="${img}")
  done
  echo -e "${GRN}Peer images:${RST} ${resolved_peer_images[*]+${resolved_peer_images[*]}}"
fi

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
pytest --reruns 2 -m "${FUNCTIONAL_TESTS_MARKER}" -c "${root_path}/pytest.ini" -n 12 \
  --dist load\
  --log-cli-level="${FUNCTIONAL_TESTS_LOG_LEVEL}" \
  --docker_project_name="${project_name}" \
  --docker-image=${image_name} \
  ${peer_image_args[@]+"${peer_image_args[@]}"} \
  --codecov_dir="${binary_coverage_reports_path}" \
  --logs-dir="${logs_path}" \
  --junitxml="${test_results_path}/report.xml" \
  ${root_path}
exit_code=$?

# Stop containers
echo -e "${GRN}Stopping docker containers${RST}"
docker compose -p ${project_name} ${all_compose_files} stop

dump_compose_logs

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
