[![Unit Tests](https://ci.status.im/buildStatus/icon?subject=Unit%20Tests&job=status-go%2Ftests-develop)](https://ci.status.im/job/status-go/job/tests-develop/) [![Unit Tests](https://ci.status.im/buildStatus/icon?subject=Functional%20Tests&job=status-go%2Ftests-rpc-develop)](https://ci.status.im/job/status-go/job/tests-rpc-develop/) [![codecov](https://codecov.io/github/status-im/status-go/graph/badge.svg?token=mQIkVPUdVv)](https://codecov.io/github/status-im/status-go)

# status-go

`status-go` is the backbone library of [Status App](https://github.com/status-im/status-app). It acts as a "backend" for both applications and implements most of the Status apps business logic. 

A comprehensive list of `status-go` functionality can be found in [Brief overview](https://www.notion.so/WIP-Brief-overview-1578f96fb65c8042a166ecd79a6b1cb6?pvs=4).

# Docs

- [How to Build](docs/building.md)
- [How to Contribute](CONTRIBUTING.md)
- [How to Release](docs/RELEASING.md)
- [How to run status-go as HTTP server](/cmd/status-backend/README.md)

# Nim token-library dependency

The optional `tkl` build uses the public
[nim-token-lists](https://github.com/status-im/nim-token-lists) Go wrapper and
native library pinned to the same revision. `make build-libtkl` prepares one
checkout under `build/deps/nim-token-lists`, including its pinned submodules.
It requires Git, Go, Nim 2.2.10 and the platform C toolchain. No separate library
checkout or local Go module replacement is needed.

An unchanged build reuses the checkout and compiled archive. A changed compiler
or build configuration rebuilds the archive. If the source pin changes, run
`make clean` to remove the managed checkout before rebuilding. Cleanup does not
remove the global Go module cache. `make test-libtkl` checks the dependency and ABI.

For controlled builds with prebuilt artifacts, supply both `NIM_TKL_INC_DIR`
(containing `tkl.h`) and `NIM_TKL_LIB_DIR` (containing the isolated `libtkl.a`).
Both must come from the recorded revision and target. The managed build supports
Linux and macOS (amd64/arm64), Windows (amd64 with MinGW), Android (arm64/amd64)
and iOS (arm64 device, arm64/amd64 simulator). Archives are kept separately under
`build/deps/nim-token-lists/build/<target>/link`; device and simulator archives
are never reused for one another. All targets share the same source checkout.

Use the existing mobile targets with `USE_NIM_TOKEN_LISTS=true`, for example:

```sh
make statusgo-android-library USE_NIM_TOKEN_LISTS=true ARCH=arm64 \
  ANDROID_NDK_ROOT="$ANDROID_NDK_ROOT" HOST_OS=darwin ANDROID_API=28
make statusgo-ios-library USE_NIM_TOKEN_LISTS=true ARCH=arm64 \
  IPHONE_SDK=iphonesimulator IOS_TARGET=14.0
```

Android uses NDK 27.2.12479018 and API 28 or newer; use `HOST_OS=linux` on Linux.
iOS requires Xcode, with minimum iOS 13 for devices and Intel simulators, or 14
for arm64 simulators. The token library uses the same SDK and minimum as the Go
consumer. The existing SDS build remains unchanged. For Linux cross builds,
supply the target `CC`, `TKL_LD`, `TKL_OBJCOPY` and `TKL_AR`; native builds use
the host tools. Windows builds run in a MinGW64 shell with `CC=gcc`.

# Optional token catalogue

The optional `tkl` build supports the C-backed Nim token catalogue. Set
`tokenListsUseNim` in the login wallet configuration to select it. The separate
`tokenListsShadow` option compares its committed catalogue with the SDK builder
using the same inputs, without another fetcher or storage writer. Both options
default to false; shadow comparison requires the Nim catalogue.

Comparison logs include revision, token differences, parser failures and timing.
They compare unique token metadata; list metadata and query behavior are outside
this comparison. Work is bounded per login to 64 snapshots or 24 hours, with
input limits and a coalescing queue. Known custom-token marker differences are
counted separately. Live development runs are still needed to assess parity.

For a shared-library build, run `make statusgo-shared-library-tkl`.
Build libsds first; the target accepts
the existing `NIM_SDS_LIB_DIR` and `NIM_SDS_INC_DIR` overrides. Set `NIM` if the
compiler is not on PATH. The target prepares the pinned native library and
reuses both its archive and libstatus when their inputs match. Native changes
invalidate Go's cgo cache; source, build-option or output changes invalidate
the backend cache. Shared-library builds keep `tkl_*` symbols private. On Windows,
managed archives also have explicit DLL export directives removed. A supplied
Windows prebuilt archive must likewise omit `.drectve` when hiding exports.
An iOS static archive is an intermediate: export hiding and verification at the
final application link remain part of the packaging gate.

`scripts/test_tkl_link.sh` verifies a small Go shared-library consumer on Linux,
macOS, Android and Windows. The Token library integration workflow checks native
consumers and DLL exports. With an existing SDS library on the compiler/linker
paths, `bash scripts/tkl_env.sh go test -tags 'tkl tkl_coexistence' ./internal/tklbuild`
also checks that both Nim runtimes can coexist. Full application packaging and
device runtime checks are still required before removing the SDK backend.

# License

[Mozilla Public License 2.0](https://github.com/status-im/status-go/blob/develop/LICENSE.md)
