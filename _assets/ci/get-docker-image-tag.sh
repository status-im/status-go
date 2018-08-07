#!/usr/bin/env bash
#
# Returns a tag for a docker image. It tries to get a tag first if availalbe,
# otherwise the first 8 characters of the HEAD git SHA is returned.

set -e -o pipefail

tag="$(git describe --exact-match --tag 2>/dev/null || git rev-parse HEAD | cut -c 1-8)"
echo $tag
