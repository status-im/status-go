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

all_compose_files="-f ${root_path}/docker-compose.anvil.yml -f ${root_path}/docker-compose.test.status-go.yml -f ${root_path}/docker-compose.waku.yml"
identifier=${BUILD_ID:-$(git rev-parse --short HEAD)}
project_name="status-go-func-tests-${identifier}"

export STATUS_BACKEND_URLS=$(eval echo http://${project_name}-status-backend-{1..${STATUS_BACKEND_COUNT}}:3333 | tr ' ' ,)

# Remove orphans
docker ps -a --filter "status-go-func-tests-${identifier}" --filter "status=exited" -q | xargs -r docker rm -f

# Run docker
echo -e "${GRN}Running tests${RST}, HEAD: $(git rev-parse HEAD)"
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
pytest --reruns 2 -m rpc -n 12 --dist loadgroup --docker_project_name=${project_name} --codecov_dir=${binary_coverage_reports_path} --junitxml=${test_results_path}/report.xml
exit_code=$?

# Stop containers
echo -e "${GRN}Stopping docker containers${RST}"
docker compose -p ${project_name} ${all_compose_files} stop

# Save logs
echo -e "${GRN}Saving logs${RST}"
docker compose -p ${project_name} ${all_compose_files} logs status-go > "${root_path}/statusd.log"
docker compose -p ${project_name} ${all_compose_files} logs status-backend > "${root_path}/status-backend.log"

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
