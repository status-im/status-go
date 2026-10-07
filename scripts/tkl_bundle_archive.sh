#!/usr/bin/env bash
# Go's c-archive mode leaves external C libraries for the final application.
# Bundle libtkl so consumers still need only libstatus and their existing libs.
set -euo pipefail
archive="${1:?Supply the Go archive and optionally its build command}"
shift
if [ "$#" -gt 0 ]; then "$@"; fi
: "${NIM_TKL_LIB_DIR:?Run through scripts/tkl_env.sh}"
test -f "$archive"
test -f "$NIM_TKL_LIB_DIR/libtkl.a"
scratch="$(mktemp -d "$(dirname "$archive")/.tkl-bundle.XXXXXX")"
trap 'rm -r -- "$scratch"' EXIT
case "$(go env GOOS)" in
  darwin|ios)
    "${TKL_LIBTOOL:-libtool}" -static -o "$scratch/bundled.a" "$archive" "$NIM_TKL_LIB_DIR/libtkl.a" ;;
  linux)
    # Simple relative names also avoid MRI quoting differences across ar versions.
    cp "$archive" "$scratch/status.a"
    cp "$NIM_TKL_LIB_DIR/libtkl.a" "$scratch/tkl.a"
    (cd "$scratch"; printf 'create bundled.a\naddlib status.a\naddlib tkl.a\nsave\nend\n' | "${TKL_AR:-ar}" -M) ;;
  *) echo "Unsupported static token-library target" >&2; exit 1 ;;
esac
mv -- "$scratch/bundled.a" "$archive"
