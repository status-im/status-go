#!/usr/bin/env bash
# One disposable checkout owned by status-go; no developer source override.
set -euo pipefail
repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source_dir="$repo/build/deps/nim-token-lists"
action="${1:-prepare}"
fail() { echo "libtkl: $*" >&2; exit 1; }
lock_native() {
  local timeout="${TKL_LOCK_TIMEOUT:-600}" deadline owner pid
  local owners
  [[ "$timeout" =~ ^[0-9]+$ ]] || fail "TKL_LOCK_TIMEOUT must be seconds (a non-negative integer)"
  deadline=$((SECONDS + 10#$timeout))
  mkdir -p "$repo/build/deps"
  lock="$repo/build/deps/.nim-token-lists-lock"
  while true; do
    if mkdir "$lock" 2>/dev/null; then
      # Use an owner-specific directory so stale waiters cannot delete a new
      # owner's PID. rmdir below only removes an empty, unowned lock.
      if mkdir "$lock/$$" 2>/dev/null; then
        owners=("$lock"/*)
        # An empty lock may be reclaimed before its creator publishes its PID.
        # If another creator replaced it, only a sole claimant may proceed.
        if [ "${#owners[@]}" -eq 1 ] && [ "${owners[0]}" = "$lock/$$" ]; then
          trap 'rmdir "$lock/$$" "$lock" 2>/dev/null || true' EXIT
          return
        fi
        rmdir "$lock/$$" 2>/dev/null || true
      fi
    fi
    for owner in "$lock"/*; do
      pid="${owner##*/}"
      [[ "$pid" =~ ^[1-9][0-9]*$ ]] || continue
      if ! kill -0 "$pid" 2>/dev/null; then
        rmdir "$owner" 2>/dev/null || true
      fi
    done
    if rmdir "$lock" 2>/dev/null; then continue; fi
    [ "$SECONDS" -lt "$deadline" ] || fail "Timed out waiting for libtkl lock after ${timeout}s ($lock)"
    sleep 1
  done
}
case "$action" in
  clean)
    lock_native
    if [ -e "$source_dir" ]; then rm -r -- "$source_dir"; fi
    exit 0 ;;
  prepare) ;;
  *) fail "Usage: $0 [prepare|clean]" ;;
esac

if [ -n "${NIM_TKL_LIB_DIR:-}${NIM_TKL_INC_DIR:-}" ]; then
  [ -n "${NIM_TKL_LIB_DIR:-}" ] && [ -n "${NIM_TKL_INC_DIR:-}" ] ||
    fail "Supply both NIM_TKL_LIB_DIR and NIM_TKL_INC_DIR for prebuilt inputs"
  [ -f "$NIM_TKL_LIB_DIR/libtkl.a" ] && [ -f "$NIM_TKL_INC_DIR/tkl.h" ] ||
    fail "Prebuilt inputs require isolated libtkl.a and tkl.h"
  exit 0
fi

pin="$(cat "$repo/scripts/tkl.version")"
if command -v sha256sum >/dev/null 2>&1; then
  tkl_sha256=(sha256sum)
elif command -v shasum >/dev/null 2>&1; then
  tkl_sha256=(shasum -a 256)
else
  fail "Install sha256sum or shasum for SHA-256 checksums"
fi
[[ "$pin" =~ ^[0-9a-f]{40}$ ]] || fail "Invalid native revision pin"
grep -Eq "github.com/status-im/nim-token-lists/go/tkl v[^[:space:]]*-${pin:0:12}([[:space:]]|$)" "$repo/go.mod" ||
  fail "Go wrapper and native revision pins disagree"
goos="$(go env GOOS)"
goarch="$(go env GOARCH)"
[ "$goos/$goarch" = "$(go env GOHOSTOS)/$(go env GOHOSTARCH)" ] ||
  fail "Cross-compilation needs target-specific prebuilt NIM_TKL_LIB_DIR/NIM_TKL_INC_DIR"
case "$goos" in
  darwin) target_os=Darwin ;;
  linux) target_os=Linux ;;
  *) fail "Unsupported native target: $goos/$goarch" ;;
esac

# Do not let two Make invocations publish or compile the same dependency at once.
lock_native
if [ ! -e "$source_dir" ]; then
  git clone --no-checkout https://github.com/status-im/nim-token-lists.git "$source_dir" >&2
  git -C "$source_dir" checkout --detach "$pin" >&2
  git -C "$source_dir" submodule update --init --recursive >&2
fi
[ "$(git -C "$source_dir" rev-parse HEAD)" = "$pin" ] ||
  fail "Managed checkout does not match the pin; run make clean and rebuild"
if git -C "$source_dir" submodule status --recursive | grep -Eq '^[+U-]'; then
  fail "Managed submodules are incomplete or at another revision; run make clean and rebuild"
fi
[ -f "$source_dir/abi/tkl.h" ] || fail "Incomplete checkout; run make clean and rebuild"
git -C "$source_dir" diff --quiet HEAD -- || fail "Modified managed sources; run make clean and rebuild"

nim="${NIM:-nim}"
cc="${CC:-$(go env CC)}"
command -v "$cc" >/dev/null || fail "CC must name one compiler executable: $cc"
case "$(basename "$cc")" in
  *clang*) nim_cc=clang ;;
  *gcc*|cc) nim_cc=gcc ;;
  *) fail "Unsupported compiler executable: $cc" ;;
esac
out="$source_dir/build"
base_cgo_cflags="${TKL_BASE_CGO_CFLAGS-${CGO_CFLAGS:-}}"
fingerprint="$(printf '%s\n' "$pin" "$goos/$goarch" "$(command -v "$nim")" \
  "$("$nim" --version)" "$cc" "$("$cc" --version)" \
  "${EXTRA_NIMFLAGS:-}" "$base_cgo_cflags" "${MACOSX_DEPLOYMENT_TARGET:-}" \
  "${SDKROOT:-}" "${TKL_ARCH:-}" "${TKL_MIN_OS:-}" "${TKL_SDK:-}" \
  "${TKL_PLATFORM:-}" "${TKL_LD:-}" "${TKL_OBJCOPY:-}" "${TKL_AR:-}" \
  "$("${tkl_sha256[@]}" "$repo/scripts/tkl_native.sh")" | "${tkl_sha256[@]}" | awk '{print $1}')"
if [ -f "$out/link/libtkl.a" ] && [ -f "$out/fingerprint" ] && [ -f "$out/archive.sha256" ] &&
   [ "$(cat "$out/fingerprint")" = "$fingerprint" ] &&
   [ "$(cat "$out/archive.sha256")" = "$("${tkl_sha256[@]}" "$out/link/libtkl.a" | awk '{print $1}')" ]; then
  exit 0
fi
if [ -d "$out" ]; then rm -r -- "$out"; fi
mkdir -p "$out/link"
flags="${EXTRA_NIMFLAGS:-} --cc:$nim_cc --$nim_cc.exe:$cc --$nim_cc.linkerexe:$cc"
for flag in $base_cgo_cflags; do flags="$flags --passC:$flag"; done
TKL_TARGET_OS="$target_os" OUT="$out" EXTRA_NIMFLAGS="$flags" \
  make -C "$source_dir" isolate NIM="$nim" >&2
cp "$out/libtkl_isolated.a" "$out/link/libtkl.a"
"${tkl_sha256[@]}" "$out/link/libtkl.a" | awk '{print $1}' > "$out/archive.sha256"
printf '%s\n' "$fingerprint" > "$out/fingerprint"
