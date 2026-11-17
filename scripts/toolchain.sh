#!/usr/bin/env bash
# Runs a command in the status-go toolchain image CI builds with; --image prints its name, --build builds it.
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# The tag follows the Dockerfile, so an edited toolchain never reuses an old image.
tag="$(git -C "$repo" hash-object Dockerfile | cut -c1-12)"
# The image runs natively, so each architecture has its own tag.
case "$(uname -m)" in
  arm64 | aarch64)
    platform=linux/arm64
    tag="$tag-arm64"
    ;;
  *) platform=linux/amd64 ;;
esac
image="ghcr.io/status-im/status-go-toolchain:$tag"

build() {
  # The build downloads the compilers; a retry continues after the layers already built.
  local attempt=1
  until docker build --platform "$platform" --target toolchain --tag "$image" "$@" "$repo"; do
    [ "$attempt" -lt 3 ] || exit 1
    attempt=$((attempt + 1))
    sleep 15
  done
}

case "${1:-}" in
  --image)
    echo "$image"
    exit 0 ;;
  --build)
    shift
    build "$@"
    exit 0 ;;
  '')
    echo "usage: $0 <command> [args...]" >&2
    exit 2 ;;
esac

if ! docker image inspect "$image" >/dev/null 2>&1 &&
  ! docker pull --quiet --platform "$platform" "$image" >/dev/null; then
  echo "toolchain: could not pull $image, building it from the Dockerfile" >&2
  build >&2
fi

cache="$HOME/.cache/status-go-toolchain"
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
# Keep the nim-sds checkout inside the mounted worktree.
args+=(--env "NIM_SDS_SOURCE_DIR=$repo/build/nim-sds")
args+=(--env "LD_LIBRARY_PATH=$repo/build/nim-sds/build:/cache/logos-storage-nim/build")
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

exec docker run "${args[@]}" "$image" "$@"
