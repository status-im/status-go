#!/usr/bin/env bash
# libtkl is an implementation detail of the Go facade, not a public libstatus ABI.
set -euo pipefail
library="${1:?Usage: check_tkl_exports.sh <libstatus>}"
case "$(uname -s)" in
  Darwin) symbols="$(nm -gU "$library")" ;;
  Linux) symbols="$(nm -D --defined-only "$library")" ;;
  *) echo "Unsupported export audit platform" >&2; exit 1 ;;
esac
leaked="$(printf '%s\n' "$symbols" | awk '$NF ~ /^_?tkl_/ {print $NF}')"
if [ -n "$leaked" ]; then
  printf 'libstatus must not export libtkl symbols:\n%s\n' "$leaked" >&2
  exit 1
fi
echo "Verified: libstatus exports no tkl_* symbols"
