#!/usr/bin/env bash
#
# Creates a string in a format: $GIT_SHA[:8][-$BUILD_TAGS]
# where $BUILD_TAGS is optional and if present all spaces
# are replaced by a hyphen (-).
#
# For example: BUILD_TAGS="tag1 tag2" ./_assets/ci/get-docker-image-tag.sh
# will produce "12345678-tag1-tag2".

set -e -o pipefail

tag="$(git rev-parse HEAD | cut -c 1-8)"

if [ ! -z "$BUILD_TAGS" ]; then
    tag="$tag-$(echo $BUILD_TAGS | sed -e "s/[[:space:]]/-/g")"
fi

echo $tag
