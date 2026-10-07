#!/usr/bin/env bash
# A C application must link a Go static archive without a separate -ltkl.
set -euo pipefail
cd "$(dirname "$0")/.."
source scripts/tkl_env.sh
mkdir -p build
go build -buildmode=c-archive -o build/tkl-static.a ./scripts/testdata/tkl-probe
bash scripts/tkl_bundle_archive.sh "$PWD/build/tkl-static.a"
flags=()
case "$(go env GOOS)" in
  darwin)
    flags=(-framework CoreFoundation -framework Security
      "-Wl,-unexported_symbols_list,$repo/scripts/tkl-unexported-macos.txt") ;;
  ios)
    arch="$(go env GOARCH)"
    [ "$arch" != amd64 ] || arch=x86_64
    minimum=13.0
    [ "${IPHONE_SDK:-iphoneos}/$arch" != iphonesimulator/arm64 ] || minimum=14.0
    triple="$arch-apple-ios${IOS_TARGET:-$minimum}"
    [ "${IPHONE_SDK:-iphoneos}" != iphonesimulator ] || triple="$triple-simulator"
    flags=(-target "$triple" -isysroot "$(xcrun --sdk "${IPHONE_SDK:-iphoneos}" --show-sdk-path)"
      -framework CoreFoundation -framework Security
      "-Wl,-unexported_symbols_list,$repo/scripts/tkl-unexported-macos.txt") ;;
  linux) flags=(-ldl "-Wl,--version-script=$repo/scripts/tkl-private-linux.map") ;;
  *) echo "Unsupported static probe target" >&2; exit 1 ;;
esac
"${CC:-cc}" scripts/testdata/tkl-caller.c -Ibuild build/tkl-static.a \
  "${flags[@]}" -pthread -lm -o build/tkl-static-app
if [ "$(go env GOOS)/$(go env GOARCH)" = "$(go env GOHOSTOS)/$(go env GOHOSTARCH)" ]; then
  ./build/tkl-static-app
fi
TKL_TARGET_OS="$(go env GOOS)" bash scripts/check_tkl_exports.sh build/tkl-static-app
