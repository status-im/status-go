#!/usr/bin/env bash
# Runs a command in the Nix image CI builds status-go-library in; --image prints its name, --build builds it.
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# What the image is built from.
inputs=(flake.nix flake.lock nix/overlay.nix nix/pkgs nix/ci.Dockerfile nix/scripts/ci-image.sh)
# The tag follows these files, so a changed dependency never reuses an old image.
tag="$(git -C "$repo" ls-tree -r HEAD -- "${inputs[@]}" |
  git hash-object --stdin | cut -c1-12)"
image="ghcr.io/status-im/status-go-nix:$tag"
platform=linux/amd64

build() {
  git -C "$repo" archive HEAD -- "${inputs[@]}" |
    docker build --platform "$platform" --file nix/ci.Dockerfile --tag "$image" -
}

case "${1:-}" in
  --image)
    echo "$image"
    exit 0 ;;
  --build)
    build
    exit 0 ;;
  '')
    echo "usage: $0 <command> [args...]" >&2
    exit 2 ;;
esac

if ! docker image inspect "$image" >/dev/null 2>&1 &&
  ! docker pull --quiet --platform "$platform" "$image" >/dev/null; then
  echo "nix-image: could not pull $image, building it (about 40 minutes)" >&2
  build >&2
fi

# The checkout keeps its host path so that git worktree pointers resolve.
args=(--rm --platform "$platform" --volume "$repo:$repo" --workdir "$repo")
git_common="$(cd "$repo" && cd "$(git rev-parse --git-common-dir)" && pwd)"
case "$git_common" in
  "$repo"/*) ;;
  *) args+=(--volume "$git_common:$git_common:ro") ;;
esac

exec docker run "${args[@]}" "$image" "$@"
