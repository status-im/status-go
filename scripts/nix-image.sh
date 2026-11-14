#!/usr/bin/env bash
# Runs a command in the Nix image CI builds the status-go-library package in.
#
#   scripts/nix-image.sh nix build --no-link '.?submodules=1#status-go-library'
#   scripts/nix-image.sh --image    # prints the image reference
#   scripts/nix-image.sh --build    # builds the image
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# What the image is built from. The hash of the Go modules changes with go.mod
# and is not part of the image.
vendor_hash=nix/pkgs/status-go/library/vendorHash.sha256
context=(flake.nix flake.lock nix ":(exclude)$vendor_hash")
# The tag follows these files, so a changed dependency never reuses an old image.
tag="$(git -C "$repo" ls-tree -r HEAD -- flake.nix flake.lock nix | grep -v "$vendor_hash" |
  git hash-object --stdin | cut -c1-12)"
image="${STATUS_GO_NIX_IMAGE:-ghcr.io/status-im/status-go-nix:$tag}"
platform=linux/amd64

build() {
  git -C "$repo" archive HEAD -- "${context[@]}" |
    docker build --platform "$platform" --file nix/ci.Dockerfile --tag "$image" "$@" -
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
  ! docker pull --quiet --platform "$platform" "$image" 2>/dev/null; then
  build
fi

# The checkout keeps its host path so that git worktree pointers resolve.
args=(--rm --platform "$platform" --volume "$repo:$repo" --workdir "$repo")
git_common="$(cd "$repo" && cd "$(git rev-parse --git-common-dir)" && pwd)"
case "$git_common" in
  "$repo"/*) ;;
  *) args+=(--volume "$git_common:$git_common:ro") ;;
esac
# The container runs as root and the checkout belongs to someone else. Nix reads
# git settings from a file only.
gitconfig="$(mktemp)"
trap 'rm -f "$gitconfig"' EXIT
printf '[safe]\n\tdirectory = *\n' > "$gitconfig"
args+=(--volume "$gitconfig:/root/.gitconfig:ro")

docker run "${args[@]}" "$image" "$@"
