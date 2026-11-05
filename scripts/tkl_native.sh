#!/usr/bin/env bash
# Builds libtkl from status-go's Nimble resolution, one output per target.
set -euo pipefail
repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
build_dir="$repo/build/tkl"
action="${1:-prepare}"
fail() { echo "libtkl: $*" >&2; exit 1; }
case "$action" in
  clean)
    if [ -e "$build_dir" ]; then rm -r -- "$build_dir"; fi
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
  [ -f "$NIM_TKL_LIB_DIR/libtkl.target" ] || fail "Prebuilt inputs require libtkl.target"
  prebuilt_target="$(<"$NIM_TKL_LIB_DIR/libtkl.target")"
  [ "$prebuilt_target" = "$archive_target" ] ||
    fail "Prebuilt libtkl target mismatch: inputs are for $prebuilt_target, requested $archive_target; unset NIM_TKL_INC_DIR and NIM_TKL_LIB_DIR to build for the target"
  printf '%s\n' "$NIM_TKL_LIB_DIR"
  exit 0
fi

pin="$(sed -n 's|^requires "https://github.com/.*/nim-token-lists#\([0-9a-f]*\)"$|\1|p' "$repo/status_go.nimble")"
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
      Darwin) host_os=darwin ;;
      Linux) host_os=linux ;;
      *) fail "Android cross builds require Linux or macOS" ;;
    esac
    sdk_identity="$(cat "$ANDROID_NDK_ROOT/source.properties" 2>/dev/null || true)"
    ;;
  ios)
    mobile=true
    case "${IPHONE_SDK:-iphoneos}" in
      iphoneos)
        [ "$arch" = arm64 ] || fail "iOS devices require arm64"
        target=ios-arm64 ;;
      iphonesimulator) target="ios-simulator-$arch" ;;
      *) fail "Unsupported IPHONE_SDK: $IPHONE_SDK" ;;
    esac
    sdk_identity="$(xcrun --sdk "${IPHONE_SDK:-iphoneos}" --show-sdk-path) $(xcrun --sdk "${IPHONE_SDK:-iphoneos}" --show-sdk-version)"
    ;;
  darwin)
    sdk_identity="$(xcrun --sdk macosx --show-sdk-path) $(xcrun --sdk macosx --show-sdk-version)"
    ;;
  linux)
    if [ "$goos/$goarch" != "$(go env GOHOSTOS)/$(go env GOHOSTARCH)" ]; then
      [ -n "${CC:-}" ] && [ -n "${TKL_LD:-}" ] && [ -n "${TKL_OBJCOPY:-}" ] && [ -n "${TKL_AR:-}" ] ||
        fail "Linux cross builds require CC, TKL_LD, TKL_OBJCOPY and TKL_AR for the target"
    fi
    ;;
  windows)
    [ "$goarch" = amd64 ] || fail "Windows requires amd64"
    "$cc" -dumpmachine | grep -Eq '^x86_64-.*mingw' || fail "Windows requires an x86_64 MinGW compiler"
    ;;
  *) fail "Unsupported native target: $goos/$goarch" ;;
esac

env -u MAKEFLAGS -u MFLAGS make --no-print-directory -C "$repo" nimble.paths >&2

# Mobile targets are spelled by nim.mk, as for libsds; desktop builds with Go's compiler.
if ! "$mobile"; then
  command -v "$cc" >/dev/null || fail "CC must name one compiler executable: $cc"
  case "$(basename "$cc")" in
    *clang*) nim_cc=clang ;;
    *gcc*|cc) nim_cc=gcc ;;
    *) fail "Unsupported compiler executable: $cc" ;;
  esac
  cc_identity="$cc $("$cc" --version)"
fi
out="$build_dir/$target"
base_cgo_cflags="${TKL_BASE_CGO_CFLAGS-${CGO_CFLAGS:-}}"
fingerprint="$(printf '%s\n' "$(cat "$repo/nimble.paths")" \
  "${cc_identity:-}" \
  "$target" "$sdk_identity" "${ANDROID_API:-28}" "${IOS_TARGET:-}" \
  "$base_cgo_cflags" "${MACOSX_DEPLOYMENT_TARGET:-}" "${TKL_LD:-}" "${TKL_OBJCOPY:-}" "${TKL_AR:-}" \
  "${NIM_PARAMS-}" "$("${tkl_sha256[@]}" "$repo/scripts/tkl_native.sh" "$repo/nim.mk")" | "${tkl_sha256[@]}" | awk '{print $1}')"
# Each build goes to its own directory and is published under its fingerprint,
# so concurrent builds of one target cannot mix their outputs.
published="$out/$fingerprint"
complete() {
  [ -f "$published/link/libtkl.a" ] && [ -f "$published/archive.sha256" ] &&
    [ "$(cat "$published/archive.sha256")" = "$("${tkl_sha256[@]}" "$published/link/libtkl.a" | awk '{print $1}')" ]
}
if complete; then
  printf '%s\n' "$published/link"
  exit 0
fi
mkdir -p "$out"
build="$(mktemp -d "$out/.build.XXXXXX")"
mkdir "$build/link"
trap 'rm -rf -- "$build"' EXIT
if "$mobile"; then
  case "$goos" in
    android) platform=(TKL_PLATFORM=ANDROID HOST_OS="$host_os") ;;
    ios) platform=(TKL_PLATFORM=IOS) ;;
  esac
  env -u MAKEFLAGS -u MFLAGS make --no-print-directory -C "$repo" nim-libtkl \
    "${platform[@]}" ARCH="$arch" NIM_LIB_DIR="$build" >&2
else
  # The target, spelled for Go's compiler; libtkl's own config has the rest.
  nim_os="$goos"
  [ "$goos" != darwin ] || nim_os=macosx
  flags="--os:$nim_os --cpu:$goarch --cc:$nim_cc --$nim_cc.exe:$cc --$nim_cc.linkerexe:$cc"
  # Linked into a shared libstatus.
  [ "$goos" = windows ] || flags="$flags --passC:-fPIC"
  [ "$goos" != darwin ] || flags="$flags --passC:-arch --passC:$arch"
  for flag in $base_cgo_cflags; do flags="$flags --passC:$flag"; done
  # A Linux or Windows cross build names the target's tools.
  [ -z "${TKL_LD:-}" ] || export NIM_SEAL_LD="$TKL_LD"
  [ -z "${TKL_OBJCOPY:-}" ] || export NIM_SEAL_OBJCOPY="$TKL_OBJCOPY"
  [ -z "${TKL_AR:-}" ] || export NIM_SEAL_AR="$TKL_AR"
  env -u MAKEFLAGS -u MFLAGS make --no-print-directory -C "$repo" nim-libtkl \
    TKL_NIM_FLAGS="$flags" NIM_LIB_DIR="$build" >&2
fi
if [ "$goos" = windows ]; then
  # Nim's dynlib declarations emit explicit dllexport directives. Linker
  # auto-export filters alone cannot make these private inside libstatus.
  "${TKL_OBJCOPY:-objcopy}" --remove-section .drectve "$build/link/libtkl.a"
fi
printf '%s\n' "$archive_target" > "$build/link/libtkl.target"
"${tkl_sha256[@]}" "$build/link/libtkl.a" | awk '{print $1}' > "$build/archive.sha256"
# A concurrent build of the same inputs may have published first; keep it.
if ! complete; then
  if [ -d "$published" ]; then rm -r -- "$published"; fi
  mv "$build" "$published" 2>/dev/null || complete || fail "Could not publish $published"
fi
for old in "$out"/*; do
  [ "$old" = "$published" ] || rm -r -- "$old"
done
printf '%s\n' "$published/link"
