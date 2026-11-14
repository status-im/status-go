#!/usr/bin/env bash
# Runs a command in the status-go toolchain image, the one CI builds with.
#
#   scripts/toolchain.sh make statusgo-library
#   scripts/toolchain.sh --image    # prints the image reference
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# The tag follows the Dockerfile, so an edited toolchain never reuses an old image.
tag="$(git -C "$repo" hash-object Dockerfile | cut -c1-12)"
# The image runs natively; only the amd64 one is published.
case "$(uname -m)" in
  arm64 | aarch64)
    platform=linux/arm64
    tag="$tag-arm64"
    ;;
  *) platform=linux/amd64 ;;
esac
image="${STATUS_GO_TOOLCHAIN_IMAGE:-ghcr.io/status-im/status-go-toolchain:$tag}"

if [ "${1:-}" = --image ]; then
  echo "$image"
  exit 0
fi
[ $# -gt 0 ] || { echo "usage: $0 <command> [args...]" >&2; exit 2; }

if ! docker image inspect "$image" >/dev/null 2>&1 &&
  ! docker pull --quiet --platform "$platform" "$image" 2>/dev/null; then
  # The build downloads the compilers; a retry continues after the layers already built.
  attempt=1
  until docker build --platform "$platform" --target toolchain --tag "$image" "$repo"; do
    [ "$attempt" -lt 3 ] || exit 1
    attempt=$((attempt + 1))
    sleep 15
  done
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
# Tests hard-link fixtures into the temp directory, so it sits on the checkout's filesystem.
mkdir -p "$repo/build/.tmp"
args+=(--env "TMPDIR=$repo/build/.tmp")
# The Makefile clones logos-storage-nim next to the checkout, which is not mounted.
args+=(--env LOGOS_STORAGE_SOURCE_DIR=/cache/logos-storage-nim)
# CI and test settings reach the container by name.
for name in $(compgen -e); do
  case "$name" in
    CI | UNIT_TEST_*) args+=(--env "$name") ;;
  esac
done
if [ -t 0 ] && [ -t 1 ]; then
  args+=(--interactive --tty)
fi

# nimble looks for a Nim under the home directory before it downloads one.
# shellcheck disable=SC2016
link_nim='if [ -d /opt/nim-sds ]; then
  version="$(/opt/nim-sds/bin/nim --version | sed -nE "1s/.*Version ([0-9.]+).*/\1/p")"
  link="$HOME/.nimble/nimbinaries/nim-$version"
  mkdir -p "$HOME/.nimble/nimbinaries"
  # A directory here is a Nim that nimble downloaded for the wrong architecture.
  if [ -d "$link" ] && [ ! -L "$link" ]; then rm -rf "$link"; fi
  ln -sfn /opt/nim-sds "$link"
fi
exec "$@"'

exec docker run "${args[@]}" "$image" bash -c "$link_nim" toolchain "$@"
