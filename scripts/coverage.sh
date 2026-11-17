#!/usr/bin/env bash

source "${GIT_ROOT}/scripts/colors.sh"

convert_coverage_to_html() {
  echo -e "${GRN}Generating HTML coverage report${RST}"

  local input_coverage_report="${1}"
  local output_coverage_report="${2}"

  go tool cover -html "${input_coverage_report}" -o "${output_coverage_report}"
}
