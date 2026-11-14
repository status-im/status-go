#!/usr/bin/env bash
# Fills the Nix store of the CI image with everything the status-go-library
# package is built from, except the Go modules and the library itself.
# Runs inside nix/ci.Dockerfile, in a copy of flake.nix, flake.lock and nix/.
set -euo pipefail

cd "$1"
roots=/nix/var/nix/gcroots/ci
mkdir -p "$roots"

# The hash of the Go modules is left out of the image, so the image survives
# a change of go.mod. Evaluation still reads the file.
vendor_hash=nix/pkgs/status-go/library/vendorHash.sha256
[ -f "$vendor_hash" ] || printf 'sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=' > "$vendor_hash"

# The package takes its version from the commit.
git init -q .
git add -A
git -c user.name=ci -c user.email=ci@localhost commit -q -m image

package='.?submodules=1#packages.x86_64-linux.status-go-library'
library="$(nix eval --raw "$package.drvPath")"
modules="$(nix eval --raw "$package.goModules.drvPath")"

# What the two derivations are built from, without the Go modules.
deps=()
while read -r drv; do
  deps+=("$drv^*")
done < <(nix-store --query --references "$library" "$modules" |
  grep '\.drv$' | grep -v -- '-go-modules\.drv$' | sort -u)
nix build -L --out-link "$roots/dep" "${deps[@]}"

# The sources of the flake inputs, which every evaluation reads.
n=0
while read -r path; do
  nix-store --add-root "$roots/input-$n" --realise "$path" > /dev/null
  n=$((n + 1))
done < <(nix flake archive --json | grep -o '/nix/store/[^"]*' | sort -u)

# Compilers and sources that only the dependencies needed.
nix-collect-garbage
du -sh /nix/store
