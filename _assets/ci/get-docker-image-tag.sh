#!/usr/bin/env bash
#
# Creates a string in a format: $GIT_SHA[:8][-$BUILD_TAGS]
# where $BUILD_TAGS is optional and if present all spaces
# are replaced by a hyphen (-).


set -e -o pipefail

tag="$(git describe --always --tag || git rev-parse HEAD | cut -c 1-8)"
echo $tag
