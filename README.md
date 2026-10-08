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
Both must come from the recorded revision and target. Native builds currently
support Linux and macOS; cross builds require matching prebuilt inputs.

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

# License

[Mozilla Public License 2.0](https://github.com/status-im/status-go/blob/develop/LICENSE.md)
