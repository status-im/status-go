#!/usr/bin/env bash

set -o nounset

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)
source "${GIT_ROOT}/_assets/scripts/colors.sh"

echo -e "${GRN}Running benchmark${RST}"

root_path="${GIT_ROOT}/tests-functional"
test_results_path="${root_path}/reports"
benchmark_results_path="${root_path}/benchmarks"

rm -rf "${test_results_path}"
rm -rf "${benchmark_results_path}"

mkdir -p "${test_results_path}"
mkdir -p "${benchmark_results_path}"

all_compose_files="-f ${root_path}/docker-compose.anvil.yml -f ${root_path}/docker-compose.waku.yml"
identifier=${BUILD_ID:-$(git rev-parse --short HEAD)}
project_name="status-go-func-tests-${identifier}"
image_name="statusgo-${identifier}"

# Remove orphans
echo -e "${GRN}Cleanup old containers${RST}"
docker ps -a --filter "name=status-go-func-tests-${identifier}" --filter "status=exited" -q | xargs -r docker rm -f

# Build statusgo image
echo -e "${GRN}Building status-go${RST}"
docker build --file ./_assets/build/Dockerfile . \
  --build-arg "enable_go_cache=false" \
  --tag "${image_name}"

# Run docker
echo -e "${GRN}Running status-go external dependencies${RST}"
docker compose -p ${project_name} ${all_compose_files} up -d --build --remove-orphans

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
echo -e "${GRN}Running benchmark${RST}, HEAD: $(git rev-parse HEAD)"
pytest \
  -m benchmark \
  -c "${root_path}/pytest.ini" \
  --log-cli-level="INFO" \
  --docker_project_name="${project_name}" \
  --docker-image="${image_name}" \
  --benchmark-results-dir="${benchmark_results_path}" \
  --junitxml="${test_results_path}/report.xml"
exit_code=$?

# Stop containers
echo -e "${GRN}Stopping docker containers${RST}"
docker compose -p ${project_name} ${all_compose_files} stop

# Cleanup containers
echo -e "${GRN}Removing docker containers${RST}"
docker compose -p ${project_name} ${all_compose_files} down

echo -e "${GRN}Testing finished${RST}"
exit $exit_code
