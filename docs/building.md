# Build status-go

## Quick start

```shell
make statusgo
```

### Run status-backend

```shell
make status-backend
```

Once that is completed, you can start it straight away by running
```shell
./build/bin/status-backend --address=localhost:12345
```

This will provide full API at http://localhost:12345. \
Checkout [`status-backend docs`](../cmd/status-backend/README.md) for more details.

## Building with your IDE

`status-go` can be built as a regular Go project. The generated sources the
library build needs are committed, so a plain `go build` works out of a fresh
clone. To regenerate them (after changing a `.proto`, a `//go:generate`
directive or a SQL migration), and to generate the test-only mocks:
- `make status-go-deps` - install required tools
- `make generate` - compile protobuf files, build SQL migrations, generate mocks

## status-go as a nimble package

status-go is also a [nimble](https://github.com/nim-lang/nimble) package
(`statusgo.nimble`), so that Nim projects — status-desktop above all — can
depend on it by revision and get both the Go sources that build `libstatus`
and the `status_go` Nim wrapper that binds to it.

### Layout

- `statusgo.nimble` — the manifest. It is purely declarative and owns the
  nim-sds pin: the Makefile derives `NIM_SDS_REPO` and `NIM_SDS_VERSION` from
  its `requires` line, so the pin has a single source of truth.
- `nimble.lock` — the resolution a status-go CHECKOUT builds against, and only
  that. nimble ignores a dependency's lock, so a consumer's own lock is the
  authority for the store copy; the two may legitimately name different
  versions of the shared dependencies.
- `statusgo.nims` — the build tasks. After a one-time `nimble setup`, run them
  as `nim <task> statusgo.nims`:

  | task | output |
  |---|---|
  | `libstatus` | static `libstatus.a` + shared `libsds` for the host |
  | `libsds` | shared `libsds` for the host |
  | `libsdsIos` | static `libsds` for iOS |
  | `libsdsAndroid` | `libsds` for Android (needs `ARCH` and `ANDROID_NDK_ROOT`) |

- `status_go.nim` + `status_go/impl.nim` — the Nim wrapper over the C API.
  `import status_go` auto-links the artifacts the `libstatus` task places in
  `build/bin`, plus the system libraries and frameworks the Go runtime needs.
  Consumers that link their own flavour of the libraries (status-desktop links
  the shared one it builds itself) compile with `-d:statusGoNoAutoLink`.
- `status_backend.nim` — a Nim host for the status-backend HTTP server,
  linking it through that same wrapper. Build it from a checkout:

  ```shell
  nim libstatus statusgo.nims
  nim c status_backend.nim
  ```

The wrapper links `libstatus.a` statically and `libsds` as a shared library,
with an rpath into `build/bin`. A fully static link is macOS-only: on Linux and
Windows nim-sds's static `libsds` exports its whole Nim runtime, which collides
with the runtime of the Nim program linking it. Only the Linux arm of the
auto-link flags is verified; macOS and Windows are by inspection.

The manifest declares no `srcDir`, so consumers get the whole tree on their
import path and every root-level `*.nim` — `status_go.nim`, `status_backend.nim`
— is an importable module name for them.

### Building from a read-only copy

A consumer resolves status-go into nimble's package store, which is a
read-only copy of this tree, and builds it there. Nothing is written into the
source tree: every output goes under one caller-chosen directory.

| variable | meaning |
|---|---|
| `STATUSGO_BUILD_DIR` | (`statusgo.nims`) root of every output. Default: the package directory |
| `STATUSGO_NIMBLE_PATHS` | (`statusgo.nims`) the `nimble.paths` to build against. Default: the one next to `statusgo.nims`. An embedder that has already resolved the graph points this at its own file |
| `NIMFLAGS` | (`statusgo.nims`) forwarded to the nim-sds compiles |
| `STATUS_GO_BUILD_DIR` | (Makefile) root of every library-target output; `STATUS_GO_BIN_DIR`, `STATUS_GO_BINDINGS_PATH`, `STATUS_GO_LIBRARY_OUT` and `STATUS_GO_STUB_BINDINGS_OUT` derive from it. Default: `./build` |
| `GENERATE_PREREQ` | (Makefile) prerequisite of the library targets, `generate` by default. Pass `GENERATE_PREREQ=` to skip regeneration |

Artifact layout under `STATUSGO_BUILD_DIR`:

```
build/bin/libstatus.*         the Go library and its generated header
build/bin/statusgo-lib/       the generated cbindings entry point
.sds-build/build/libsds.*     the nim-sds artifacts
.sds-build/library/libsds.h   nim-sds's header contract
```

### Generated Go sources are committed

Everything the LIBRARY targets need from `make generate` is committed, so the
store copy builds with no generator toolchain anywhere near it:

`*.pb.go`, `bindata.go`, `migrations.go`,
`cmd/status-backend/server/endpoints.go` and
`internal/protocol/messenger_handlers.go`. The mocks are test-only and stay
untracked.

Regenerate them with `make generate` from a checkout that has the toolchain
(protoc, mockgen, `go tool go-generate-fast`) and commit the diff. CI runs
`make generate` and then `git diff --exit-code`, so drift fails the PR.
`scripts/cleanup_generated_files.sh` sweeps only the untracked, test-only
mocks.

### Build values come from `-ldflags`

`pkg/version` and `pkg/sentry` take their build-time values as plain package
variables set at link time (`BUILD_VARS_LDFLAGS` in the Makefile), not from
generated files:

| linker variable | source |
|---|---|
| `pkg/version.version` | `STATUS_GO_VERSION`; defaults to `git describe --tags`, else `0.0.0-dev` |
| `pkg/version.gitCommit` | `GIT_COMMIT`; defaults to `git rev-parse --short HEAD`, else `unknown` |
| `pkg/sentry.defaultContextName` | `SENTRY_CONTEXT_NAME` |
| `pkg/sentry.defaultContextVersion` | `SENTRY_CONTEXT_VERSION`; defaults to `STATUS_GO_VERSION` |
| `pkg/sentry.production` | `SENTRY_PRODUCTION` |

A package-store copy is not a git checkout, so the git-derived defaults fall
back to the placeholders above; an embedder that knows the real version passes
`STATUS_GO_VERSION` in. In the shipping configuration it does: status-desktop
passes its OWN version, so `pkg/version.Version()` inside libstatus reports the
app's version, not status-go's.

Extra link flags belong in `GO_EXTRA_LDFLAGS`. Overriding `BUILD_FLAGS`
replaces `BUILD_VARS_LDFLAGS` and leaves the build unstamped.

The reproducibility flags (`-buildid=`, the `ZERO_AR_DATE` repack) are on the
mobile targets only. The desktop targets do not get them: the desktop consumer
gates a relink on the artifact existing, not on its bytes.

## Building with Docker

```shell
docker build .
```

## Building using Nix shell

It is advised but not required to use Nix shell before executing other make targets. Nix shell will ensure that all dependencies are installed.
You can enter the development shell by using either of two:
```bash
make shell
nix develop --extra-experimental-features 'nix-command flakes'
```

### Build a library for the current platform

```shell
make statusgo-library      # Build static library
make statusgo-shared-library # Build shared library
```

## Build status-go with nix

The `flake.nix` file exposes multiple `status-go` packages that can be built using Nix. To view the available packages for different architectures, run:
```bash
nix flake show
```
To build a specific package, use:
```bash
nix build '#name-of-the-package'
```
For example:
```bash
nix build '#status-go-library'
```
This flake includes a dependency on `nwaku`, which is pinned to a specific commit to ensure reproducibility and control over its version.
Maintainers are responsible for tracking updates to `nwaku` and updating the pinned commit accordingly when new versions are released.
Continuous Integration (CI) will validate whether the `status-go` packages build successfully with Nix, and report the result on pull requests.

## Debugging

### Android debugging

In order to see the log files while debugging on an Android device, do the following:

* Ensure that the app can write to disk by granting it file permissions. For that, you can for instance set your avatar from a file on disk.
* Connect a USB cable to your phone and make sure you can use adb.
Run

```shell
adb shell tail -f sdcard/Android/data/im.status.ethereum.debug/files/Download/geth.log
```

## Linting

```shell
make lint
```

## Testing

Next, run unit tests:

```shell
make test
```

Unit tests can also be run using `go test` command. If you want to launch specific test, for instance `RPCSendTransactions`, use the following command:

```shell
go test -v ./api/ -testify.m ^RPCSendTransaction$
```

Or use `make test-single`:

```
make test-single PKG=./messaging/controller/processor TEST=^TestSDSWrappedMessages$
```

Note `-testify.m` as [testify/suite](https://godoc.org/github.com/stretchr/testify/suite) is used to group individual tests.

To run a single test in a test suite (e.g. `TestTransferringKeystoreFiles`, which is part of `SyncDeviceSuite`):
```shell
go test -v ./server/pairing -test.run TestSyncDeviceSuite -testify.m ^TestTransferringKeystoreFiles$
```

Or with `make test-single`:
```shell
make test-single PKG=./server/pairing TEST=TestSyncDeviceSuite TESTIFY_M=^TestTransferringKeystoreFiles$
```

Note: `TestSyncDeviceSuite` is not the name of the test suite, but the name of the test function that runs the `SyncDeviceSuite` suite.
