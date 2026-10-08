#!/usr/bin/env bash
# Reuse a matching tagged backend. Invoked after cached native preparation.
set -euo pipefail
if command -v sha256sum >/dev/null 2>&1; then
  tkl_sha256=(sha256sum)
elif command -v shasum >/dev/null 2>&1; then
  tkl_sha256=(shasum -a 256)
else
  echo "libtkl: Install sha256sum or shasum for SHA-256 checksums" >&2
  exit 1
fi
cd "$(dirname "${BASH_SOURCE[0]}")/.."
library="${1:?Supply the output library and build command}"
shift
[ "$#" -gt 0 ] || exit 2
mkdir -p "$(dirname "$library")"
stamp="$library.tkl-inputs"
lock="$library.tkl-lock"
mkdir "$lock" 2>/dev/null || { echo "Another tagged backend build is active" >&2; exit 1; }
trap 'rmdir "$lock"' EXIT

inputs="$({
  git rev-parse HEAD
  go env GOOS GOARCH GOVERSION CGO_ENABLED CC GOFLAGS GOAMD64 GOARM64
  printf '%s\n' "${CGO_CFLAGS:-}" "${CGO_LDFLAGS:-}" "${CGO_CPPFLAGS:-}" \
    "${CGO_CXXFLAGS:-}" "${GOEXPERIMENT:-}" "${MACOSX_DEPLOYMENT_TARGET:-}" "${SDKROOT:-}"
  printf '%s\n' "${BUILD_FLAGS:-}" "${SENTRY_CONTEXT_NAME:-}" \
    "${SENTRY_CONTEXT_VERSION:-}" "${SENTRY_PRODUCTION:-}"
  # Command-line Make overrides can affect generation and linking. Jobserver
  # descriptors vary between invocations without changing the resulting binary.
  printf '%s\n' "${MAKEFLAGS:-}" | sed -E 's/--jobserver[^ ]*//g'
  printf '%s\0' "$@"
  # Include local source edits and new files; generated/compiled outputs are
  # ignored by Git and cannot make an unchanged build invalidate itself.
  git ls-files -z --cached --others --exclude-standard |
    while IFS= read -r -d '' file; do
      if [ -f "$file" ]; then printf '%s\0' "$file"; fi
    done | xargs -0 "${tkl_sha256[@]}"
} | "${tkl_sha256[@]}" | awk '{print $1}')"

if [ -f "$library" ] && [ -f "$stamp" ]; then
  expected="$inputs $("${tkl_sha256[@]}" "$library" | awk '{print $1}')"
  if [ "$(cat "$stamp")" = "$expected" ]; then
    echo "Reusing tagged backend: $library"
    exit 0
  fi
fi
# Never retain a success marker for a failed or interrupted build.
if [ -f "$stamp" ]; then rm -- "$stamp"; fi
"$@"
printf '%s %s\n' "$inputs" "$("${tkl_sha256[@]}" "$library" | awk '{print $1}')" > "$stamp"
