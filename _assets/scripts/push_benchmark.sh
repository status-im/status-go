#!/usr/bin/env bash
set -o nounset
set -o errexit
set -o pipefail

REPO_URL="git@github.com:status-im/status-go-benchmarks.git"

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)
source "${GIT_ROOT}/_assets/scripts/colors.sh"

echo -e "${GRN}Pushing benchmark results${RST}"

cd "${GIT_ROOT}"
# Get the commit SHA from the status-go repo BEFORE cloning bench-repo
commit_sha=$(git rev-parse --short HEAD)

git clone "${REPO_URL}" benchmarks-repo
cd benchmarks-repo

timestamp=$(date -u '+%Y%m%dT%H%M%S')
benchmark_dir="${timestamp}_${commit_sha}"

echo -e "${GRN}Creating benchmark directory${RST}"
mkdir -p "benchmarks/${benchmark_dir}"

echo -e "${GRN}Copying benchmark results${RST}"
cp -r "${GIT_ROOT}/tests-functional/.results/benchmarks"/* "benchmarks/${benchmark_dir}/"

echo -e "${GRN}Creating virtual environment${RST}"
python3 -m venv .venv
source .venv/bin/activate

echo -e "${GRN}Installing dependencies${RST}"
pip install --upgrade pip
pip install -r requirements.txt

echo -e "${GRN}Updating README${RST}"
python ./scripts/update_readme.py

echo -e "${GRN}Committing changes${RST}"
git add .
git commit -m "Add benchmark results ${benchmark_dir}"

echo -e "${GRN}Pushing changes${RST}"
git push "${REPO_URL}"

echo -e "${GRN}Push finished${RST}"
