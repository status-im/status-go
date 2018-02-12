#!/usr/bin/env bash
#
# Description

set -e -o pipefail

tag="$(git rev-parse HEAD | cut -c 1-8)"

if [ ! -z "$BUILD_TAGS" ]; then
    tag="$tag-$(echo $BUILD_TAGS | sed -e "s/[[:space:]]/-/g")"
fi

echo $tag
