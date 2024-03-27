#!/usr/bin/env bash
set -e

RED=$(tput -Txterm setaf 1)
RST=$(tput -Txterm sgr0)

# For regex matching.
NEWLINE='
'

GIT_TAGS=$(git tag --points-at HEAD)

if [[ "${GIT_TAGS}" =~ .*${NEWLINE}.* ]]; then
  echo "${RED}Multiple tags detected and ignored!${RST}" >&2
  GIT_TAGS=''
fi

if [[ -z "${GIT_TAGS}" ]]; then
  echo "v$(git show -s --format=%as-%h)"
else
  echo "${GIT_TAGS}"
fi
