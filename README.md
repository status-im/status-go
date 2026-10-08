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

Status-go uses the public
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

Use the existing mobile targets, for example:

```sh
make statusgo-android-library ARCH=arm64 \
  ANDROID_NDK_ROOT="$ANDROID_NDK_ROOT" HOST_OS=darwin ANDROID_API=28
make statusgo-ios-library ARCH=arm64 \
  IPHONE_SDK=iphonesimulator IOS_TARGET=14.0
```

Android uses NDK 27.2.12479018 and API 28 or newer; use `HOST_OS=linux` on Linux.
iOS requires Xcode, with minimum iOS 13 for devices and Intel simulators, or 14
for arm64 simulators. The token library uses the same SDK and minimum as the Go
consumer. The existing SDS build remains unchanged. For Linux cross builds,
supply the target `CC`, `TKL_LD`, `TKL_OBJCOPY` and `TKL_AR`; native builds use
the host tools. Windows builds run in a MinGW64 shell with `CC=gcc`.

# Token catalogue

The C-backed Nim catalogue is the sole token backend. Normal executable,
shared/static library, mobile, test and lint Make targets prepare its native
dependency automatically. The `tkl` build tag is no longer needed. For direct
Go commands, use `bash scripts/tkl_env.sh go test ...` or source that script
in the same shell first (it enables strict shell mode).

Wallet consumers and the native facade use status-go's token/list DTOs in
`pkg/services/wallet/token/tokenlist`. Their JSON fields, token-key format and
database schema remain unchanged. The offline list analyzer uses the same C
facade. No packages under `go-wallet-sdk/pkg/tokens` are used; unrelated SDK
packages remain dependencies.

The `tokenListsUseNim` and `tokenListsShadow` configuration fields have been
removed. Older JSON containing those names is decoded with them ignored. There is no runtime SDK
rollback or shadow comparison. A rollback requires rebuilding an earlier version.

For a shared-library build, run `make statusgo-shared-library`.
The target retains the existing libsds preparation and accepts
the `NIM_SDS_LIB_DIR` and `NIM_SDS_INC_DIR` overrides for prebuilt inputs. Set `TKL_NIM` to
select a compiler for token lists only, or `NIM` when sharing the compiler with
other dependencies. The target prepares the pinned native library and
reuses both its archive and libstatus when their inputs match. Native changes
invalidate Go's cgo cache; source, build-option or output changes invalidate
the backend cache. Shared-library builds keep `tkl_*` symbols private. On Windows,
managed archives also have explicit DLL export directives removed. A supplied
Windows prebuilt archive must likewise omit `.drectve` when hiding exports.
Static-library builds bundle libtkl into libstatus.a, so the application
does not need a separate token-library linker input. Export hiding and verification
still happen at the final application link.

`scripts/test_tkl_link.sh` verifies a small Go shared-library consumer on Linux,
macOS, Android and Windows. The Token library integration workflow checks native
consumers and DLL exports. `scripts/test_tkl_static.sh` links a C application to
the bundled Go archive on Linux, macOS, iOS and Windows, checks private exports,
and runs it on the native host. With an existing SDS library on the compiler/linker
paths, `bash scripts/tkl_env.sh go test -tags tkl_coexistence ./internal/tklbuild`
also checks that both Nim runtimes can coexist. Full application packaging and
device runtime checks remain part of release validation.

# License

[Mozilla Public License 2.0](https://github.com/status-im/status-go/blob/develop/LICENSE.md)
