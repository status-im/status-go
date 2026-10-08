#!/usr/bin/env bash
# Test the pinned public token library through the C ABI.
# TKL_RACE=0 runs production benchmarks, e.g.:
# TKL_RACE=0 bash scripts/test_tkl.sh -run '^$' -bench Mirror -benchmem ./pkg/services/wallet/token/tklmanager
# Existing libsds builds use the same NIM_SDS_* directory overrides as Makefile.
set -euo pipefail
repo="$(cd "$(dirname "$0")/.." && pwd)"
sds_source="${NIM_SDS_SOURCE_DIR:-$repo/../nim-sds}"
sds_include="${NIM_SDS_INC_DIR:-$sds_source/library}"
sds_lib="${NIM_SDS_LIB_DIR:-$sds_source/build}"
if [ ! -f "$sds_include/libsds.h" ]; then
  echo "Missing libsds.h; set NIM_SDS_INC_DIR to the libsds include directory." >&2
  exit 1
fi
if [ ! -f "$sds_lib/libsds.a" ] && [ ! -f "$sds_lib/libsds.so" ] && [ ! -f "$sds_lib/libsds.dylib" ]; then
  echo "Build libsds first or set NIM_SDS_LIB_DIR to its library directory." >&2
  exit 1
fi
case "${TKL_RACE:-1}" in
  1) race=true ;;
  0) race=false ;;
  *) echo "TKL_RACE must be 0 or 1" >&2; exit 1 ;;
esac
source "$repo/scripts/tkl_env.sh"
export CGO_CFLAGS="${CGO_CFLAGS:-} -I$sds_include"
export CGO_LDFLAGS="${CGO_LDFLAGS:-} -L$sds_lib -lsds -Wl,-rpath,$sds_lib"
cd "$repo"
if [ "$#" -eq 0 ]; then
  set -- ./pkg/services/wallet/token/...
fi
go test -race="$race" -tags 'tkl gowaku_no_rln' "$@"
