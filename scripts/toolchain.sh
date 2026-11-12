#!/usr/bin/env bash
# Runs a command in the status-go toolchain image, the one CI builds with.
#
#   scripts/toolchain.sh make statusgo-library
#   scripts/toolchain.sh --image    # prints the image reference
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# The tag follows the Dockerfile, so an edited toolchain never reuses an old image.
tag="$(git -C "$repo" hash-object Dockerfile | cut -c1-12)"
image="${STATUS_GO_TOOLCHAIN_IMAGE:-ghcr.io/status-im/status-go-toolchain:$tag}"
# libsds does not build for linux/arm64.
platform=linux/amd64

if [ "${1:-}" = --image ]; then
  echo "$image"
  exit 0
fi
[ $# -gt 0 ] || { echo "usage: $0 <command> [args...]" >&2; exit 2; }

if ! docker image inspect "$image" >/dev/null 2>&1; then
  docker pull --quiet --platform "$platform" "$image" 2>/dev/null ||
    docker build --platform "$platform" --target toolchain --tag "$image" "$repo"
fi

cache="${STATUS_GO_TOOLCHAIN_CACHE:-$HOME/.cache/status-go-toolchain}"
mkdir -p "$cache/home" "$cache/go-build" "$cache/go-mod"

# The checkout keeps its host path so that git worktree pointers resolve.
args=(--rm --platform "$platform" --user "$(id -u):$(id -g)")
args+=(--volume "$repo:$repo" --workdir "$repo" --volume "$cache:/cache")
git_common="$(cd "$repo" && cd "$(git rev-parse --git-common-dir)" && pwd)"
case "$git_common" in
  "$repo"/*) ;;
  *) args+=(--volume "$git_common:$git_common:ro") ;;
esac
args+=(--env HOME=/cache/home --env GOCACHE=/cache/go-build --env GOMODCACHE=/cache/go-mod)
# Tools built during the build link against the libsds.so it produces.
args+=(--env "LD_LIBRARY_PATH=$repo/build")
# The Makefile clones logos-storage-nim next to the checkout, which is not mounted.
args+=(--env LOGOS_STORAGE_SOURCE_DIR=/cache/logos-storage-nim)
# CI and test settings reach the container by name.
for name in $(compgen -e); do
  case "$name" in
    CI | CODECOV_* | UNIT_TEST_*) args+=(--env "$name") ;;
  esac
done
if [ -t 0 ] && [ -t 1 ]; then
  args+=(--interactive --tty)
fi

exec docker run "${args[@]}" "$image" "$@"
