#!/usr/bin/env bash
# Run by nix/ci.Dockerfile: fills the Nix store with what status-go-library is built from, except the Go modules.
set -euo pipefail

cd "$1"
roots=/nix/var/nix/gcroots/ci
mkdir -p "$roots"

# The package takes its version from the commit.
git init -q .
git add -A
git -c user.name=ci -c user.email=ci@localhost commit -q -m image

# Fetching the flake inputs prints every remote ref, enough to cut the build log short; shown only on failure.
quiet() {
  "$@" 2> /tmp/stderr.log || { tail -n 50 /tmp/stderr.log >&2; return 1; }
}

# The sources of the flake inputs, which every evaluation reads.
inputs="$(quiet nix flake archive --json | grep -o '/nix/store/[^"]*' | sort -u)"
n=0
while read -r path; do
  nix-store --add-root "$roots/input-$n" --realise "$path" > /dev/null
  n=$((n + 1))
done <<< "$inputs"

package='.?submodules=1#packages.x86_64-linux.status-go-library'
library="$(quiet nix eval --raw "$package.drvPath")"
modules="$(quiet nix eval --raw "$package.goModules.drvPath")"

# What the two derivations are built from, without the Go modules.
drvs="$(nix-store --query --references "$library" "$modules" |
  grep '\.drv$' | grep -v -- '-go-modules\.drv$' | sort -u)"
deps=()
while read -r drv; do
  deps+=("$drv^*")
done <<< "$drvs"
nix build --max-jobs auto --out-link "$roots/dep" "${deps[@]}"

# Compilers and sources that only the dependencies needed.
nix-collect-garbage
du -sh /nix/store
