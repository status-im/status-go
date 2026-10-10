#!/usr/bin/env bash
# Checks that internal/crypto/argon2 is exactly the upstream files listed in its UPSTREAM file,
# at the golang.org/x/crypto version go.mod uses, plus argon2.patch. Any other change fails.
set -euo pipefail

GIT_ROOT=$(cd "${BASH_SOURCE%/*}" && git rev-parse --show-toplevel)
FORK="${GIT_ROOT}/internal/crypto/argon2"
cd "${GIT_ROOT}"

module=$(awk '$1 == "module" { print $2 }' "${FORK}/UPSTREAM")
version=$(awk '$1 == "version" { print $2 }' "${FORK}/UPSTREAM")
gomod_version=$(go list -m -f '{{.Version}}' "${module}")
if [[ "${version}" != "${gomod_version}" ]]; then
  echo "internal/crypto/argon2 is forked from ${module} ${version}, but go.mod uses ${gomod_version}." >&2
  echo "Re-sync the fork: copy the files listed in UPSTREAM from ${gomod_version}, re-apply argon2.patch, bump UPSTREAM." >&2
  exit 1
fi

src=$(go mod download -json "${module}@${version}" | sed -n 's/^[[:space:]]*"Dir": "\(.*\)",$/\1/p')
[[ -d "${src}" ]] || { echo "cannot locate ${module}@${version} in the module cache" >&2; exit 1; }

tmp=$(mktemp -d)
trap 'rm -rf "${tmp}"' EXIT
while read -r kind from to; do
  [[ "${kind}" == "file" ]] || continue
  cp "${src}/${from}" "${tmp}/${to}"
done < "${FORK}/UPSTREAM"
chmod -R u+w "${tmp}"
patch --quiet --no-backup-if-mismatch -p1 -d "${tmp}" < "${FORK}/argon2.patch"

if ! diff -r -x '*_test.go' -x UPSTREAM -x argon2.patch -x .gitattributes "${tmp}" "${FORK}"; then
  echo "internal/crypto/argon2 differs from ${module}@${version} + argon2.patch (diff above)." >&2
  echo "Fold intended changes into argon2.patch (diff -u against the upstream file)." >&2
  exit 1
fi
echo "internal/crypto/argon2 matches ${module}@${version} + argon2.patch"
