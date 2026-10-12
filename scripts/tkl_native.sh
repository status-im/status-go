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

goos="$(go env GOOS)"
goarch="$(go env GOARCH)"
archive_target="$goos/$goarch"
if [ "$goos" = ios ]; then
  archive_target="$archive_target/${IPHONE_SDK:-iphoneos}"
fi

if [ -n "${NIM_TKL_LIB_DIR:-}${NIM_TKL_INC_DIR:-}" ]; then
  [ -n "${NIM_TKL_LIB_DIR:-}" ] && [ -n "${NIM_TKL_INC_DIR:-}" ] ||
    fail "Supply both NIM_TKL_LIB_DIR and NIM_TKL_INC_DIR for prebuilt inputs"
  [ -f "$NIM_TKL_LIB_DIR/libtkl.a" ] && [ -f "$NIM_TKL_INC_DIR/tkl.h" ] ||
    fail "Prebuilt inputs require isolated libtkl.a and tkl.h"
  if [ -f "$NIM_TKL_LIB_DIR/libtkl.target" ]; then
    prebuilt_target="$(<"$NIM_TKL_LIB_DIR/libtkl.target")"
  else
    # Preserve native prebuilt inputs predating target metadata. Cross builds
    # must declare their target rather than silently assuming a host archive.
    prebuilt_target="$(go env GOHOSTOS)/$(go env GOHOSTARCH)"
  fi
  [ "$prebuilt_target" = "$archive_target" ] ||
    fail "Prebuilt libtkl target mismatch: inputs are for $prebuilt_target, requested $archive_target; unset NIM_TKL_INC_DIR and NIM_TKL_LIB_DIR to build for the target"
  if [ "${TKL_HIDE_EXPORTS:-0}" = 1 ] && [ "$goos" = windows ]; then
    sections="$("${OBJDUMP:-objdump}" -h "$NIM_TKL_LIB_DIR/libtkl.a")"
    if printf '%s\n' "$sections" | grep '\.drectve' >/dev/null; then
      fail "Prebuilt Windows archive still contains DLL export directives; remove .drectve from a private copy first"
    fi
  fi
  printf '%s\n' "$NIM_TKL_LIB_DIR"
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
case "$goarch" in
  arm64) arch=arm64 ;;
  amd64) arch=x86_64 ;;
  *) fail "Unsupported architecture: $goos/$goarch" ;;
esac
target="$goos-$goarch"
mobile=false
sdk_identity=""
cc="${CC:-$(go env CC)}"
case "$goos" in
  android)
    mobile=true
    target="android-$arch"
    : "${ANDROID_NDK_ROOT:?Set ANDROID_NDK_ROOT to the Android NDK}"
    case "$(uname -s)" in
      Darwin) host=darwin-x86_64 ;;
      Linux) host=linux-x86_64 ;;
      *) fail "Android cross builds require Linux or macOS" ;;
    esac
    triple=aarch64-linux-android
    [ "$goarch" != amd64 ] || triple=x86_64-linux-android
    cc="$ANDROID_NDK_ROOT/toolchains/llvm/prebuilt/$host/bin/$triple${ANDROID_API:-28}-clang"
    sdk_identity="$(cat "$ANDROID_NDK_ROOT/source.properties" 2>/dev/null || true)"
    ;;
  ios)
    mobile=true
    # Clang accepts "17", but ld -platform_version requires "17.0".
    if [[ "${IOS_TARGET:-}" =~ ^[0-9]+$ ]]; then
      export IOS_TARGET="$IOS_TARGET.0"
    fi
    case "${IPHONE_SDK:-iphoneos}" in
      iphoneos)
        [ "$arch" = arm64 ] || fail "iOS devices require arm64"
        target=ios-arm64 ;;
      iphonesimulator) target="ios-simulator-$arch" ;;
      *) fail "Unsupported IPHONE_SDK: $IPHONE_SDK" ;;
    esac
    cc="$(xcrun --sdk "${IPHONE_SDK:-iphoneos}" --find clang)"
    sdk_identity="$(xcrun --sdk "${IPHONE_SDK:-iphoneos}" --show-sdk-path) $(xcrun --sdk "${IPHONE_SDK:-iphoneos}" --show-sdk-version)"
    ;;
  darwin)
    target_os=Darwin
    export TKL_ARCH="$arch"
    sdk_identity="$(xcrun --sdk macosx --show-sdk-path) $(xcrun --sdk macosx --show-sdk-version)"
    ;;
  linux)
    target_os=Linux
    if [ "$goos/$goarch" != "$(go env GOHOSTOS)/$(go env GOHOSTARCH)" ]; then
      [ -n "${CC:-}" ] && [ -n "${TKL_LD:-}" ] && [ -n "${TKL_OBJCOPY:-}" ] && [ -n "${TKL_AR:-}" ] ||
        fail "Linux cross builds require CC, TKL_LD, TKL_OBJCOPY and TKL_AR for the target"
    fi
    ;;
  windows)
    [ "$goarch" = amd64 ] || fail "Windows requires amd64"
    target_os=Windows
    "$cc" -dumpmachine | grep -Eq '^x86_64-.*mingw' || fail "Windows requires an x86_64 MinGW compiler"
    ;;
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

# A token-only compiler override leaves other Nim dependencies' toolchains alone.
nim="${TKL_NIM:-${NIM:-nim}}"
command -v "$cc" >/dev/null || fail "CC must name one compiler executable: $cc"
case "$(basename "$cc")" in
  *clang*) nim_cc=clang ;;
  *gcc*|cc) nim_cc=gcc ;;
  *) fail "Unsupported compiler executable: $cc" ;;
esac
out="$source_dir/build/$target"
base_cgo_cflags="${TKL_BASE_CGO_CFLAGS-${CGO_CFLAGS:-}}"
fingerprint="$(printf '%s\n' "$pin" "$goos/$goarch" "$(command -v "$nim")" \
  "$("$nim" --version)" "$cc" "$("$cc" --version)" \
  "$target" "$sdk_identity" "${ANDROID_API:-28}" "${IOS_TARGET:-}" \
  "${EXTRA_NIMFLAGS:-}" "$base_cgo_cflags" "${MACOSX_DEPLOYMENT_TARGET:-}" \
  "${SDKROOT:-}" "${TKL_ARCH:-}" "${TKL_MIN_OS:-}" "${TKL_SDK:-}" \
  "${TKL_PLATFORM:-}" "${TKL_LD:-}" "${TKL_OBJCOPY:-}" "${TKL_AR:-}" \
  "$("${tkl_sha256[@]}" "$repo/scripts/tkl_native.sh")" | "${tkl_sha256[@]}" | awk '{print $1}')"
if [ -f "$out/link/libtkl.a" ] && [ -f "$out/fingerprint" ] && [ -f "$out/archive.sha256" ] &&
   [ -f "$out/link/libtkl.target" ] && [ "$(<"$out/link/libtkl.target")" = "$archive_target" ] &&
   [ "$(cat "$out/fingerprint")" = "$fingerprint" ] &&
   [ "$(cat "$out/archive.sha256")" = "$("${tkl_sha256[@]}" "$out/link/libtkl.a" | awk '{print $1}')" ]; then
  printf '%s\n' "$out/link"
  exit 0
fi
if [ -d "$out" ]; then rm -r -- "$out"; fi
mkdir -p "$out/link"
if "$mobile"; then
  OUT="build/$target" NIM="$nim" TKL_BUILD_TESTS=0 \
    bash "$source_dir/scripts/build_mobile.sh" "$target" >&2
else
  nim_os="$goos"
  [ "$goos" != darwin ] || nim_os=macosx
  flags="${EXTRA_NIMFLAGS:-} --os:$nim_os --cpu:$goarch --cc:$nim_cc --$nim_cc.exe:$cc --$nim_cc.linkerexe:$cc"
  [ "$goos" = windows ] || flags="$flags --passC:-fPIC"
  [ "$goos" != darwin ] || flags="$flags --passC:-arch --passC:$arch"
  for flag in $base_cgo_cflags; do flags="$flags --passC:$flag"; done
  TKL_TARGET_OS="$target_os" OUT="build/$target" EXTRA_NIMFLAGS="$flags" \
    make -C "$source_dir" isolate NIM="$nim" >&2
  cp "$out/libtkl_isolated.a" "$out/link/libtkl.a"
  if [ "$goos" = windows ]; then
    # Nim's dynlib declarations emit explicit dllexport directives. Linker
    # auto-export filters alone cannot make these private inside libstatus.
    "${TKL_OBJCOPY:-objcopy}" --remove-section .drectve "$out/link/libtkl.a"
  fi
fi
printf '%s\n' "$archive_target" > "$out/link/libtkl.target"
"${tkl_sha256[@]}" "$out/link/libtkl.a" | awk '{print $1}' > "$out/archive.sha256"
printf '%s\n' "$fingerprint" > "$out/fingerprint"
printf '%s\n' "$out/link"
