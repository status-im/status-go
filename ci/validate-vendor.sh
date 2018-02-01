#!/usr/bin/env bash
# Copyright 2017 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.
#
# This script checks if we changed anything with regard to dependency management
# for our repo and makes sure that it was done in a valid way.
#
# This file is a copy of https://github.com/golang/dep/blob/master/hack/validate-vendor.bash
# with some comments added.

set -e -o pipefail

# Is validate upstream empty ?
if [ -z "$VALIDATE_UPSTREAM" ]; then
	VALIDATE_REPO='https://github.com/status-im/status-go'
	VALIDATE_BRANCH='develop'

	VALIDATE_HEAD="$(git rev-parse --verify HEAD)"

	git fetch -q "$VALIDATE_REPO" "refs/heads/$VALIDATE_BRANCH"
	VALIDATE_UPSTREAM="$(git rev-parse --verify FETCH_HEAD)"

	VALIDATE_COMMIT_DIFF="$VALIDATE_UPSTREAM...$VALIDATE_HEAD"

	validate_diff() {
		if [ "$VALIDATE_UPSTREAM" != "$VALIDATE_HEAD" ]; then
		    git diff "$VALIDATE_COMMIT_DIFF" "$@"
		fi
	}
fi

IFS=$'\n'
files=( $(validate_diff --diff-filter=ACMR --name-only -- 'Gopkg.toml' 'Gopkg.lock' 'vendor/' || true) )
unset IFS

# `files[@]` splits the content of files by whitespace and returns a list.
# `#` returns the number of the lines.
if [ ${#files[@]} -gt 0 ]; then
	dep ensure -vendor-only

	# Let see if the working directory is clean
	diffs="$(git status --porcelain -- vendor Gopkg.toml Gopkg.lock 2>/dev/null)"
	if [ "$diffs" ]; then
		{
			echo 'The contents of vendor differ after "dep ensure":'
			echo
			echo "$diffs"
			echo
			echo 'Make sure these commands have been run before committing.'
			echo
		} >&2
		false
	else
		echo 'Congratulations! All vendoring changes are done the right way.'
	fi
else
    echo 'No vendor changes in diff.'
fi
