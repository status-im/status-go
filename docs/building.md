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

`status-go` can be build as a regular Go project, but requires to pre-generate some files first:
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
- `statusgo.nims` — the nim-sds build tasks. After a one-time `nimble setup`,
  run them as `nim <task> statusgo.nims`:

  | task | output |
  |---|---|
  | `libsds` | shared `libsds` for the host |
  | `libsdsIos` | static `libsds` for iOS |
  | `libsdsAndroid` | `libsds` for Android (needs `ARCH` and `ANDROID_NDK_ROOT`; runs nim-sds's per-CPU task) |

  The Go library is still built by the Makefile: `make statusgo-shared-library
  NIM_SDS_LIB_DIR=… NIM_SDS_INC_DIR=…` pointed at those artifacts.
- `status_go.nim` + `status_go/impl.nim` — the Nim wrapper over the C API. It
  declares the imports only; linking `libstatus` (and `libsds`, and the system
  libraries the Go runtime needs) into the final executable is the consumer's
  job, exactly as it was when the wrapper lived in status-desktop.

The manifest declares no `srcDir`, so consumers get the whole tree on their
import path and `status_go` is the importable module name.

### Building from a read-only copy

A consumer resolves status-go into nimble's package store, which is a
read-only copy of this tree, and builds it there. Nothing is written into the
source tree: every output goes under one caller-chosen directory.

| variable | meaning |
|---|---|
| `STATUSGO_BUILD_DIR` | (`statusgo.nims`) root of every output. Default: the package directory |
| `STATUSGO_NIMBLE_PATHS` | (`statusgo.nims`) the `nimble.paths` to build against. Default: the one next to `statusgo.nims`. An embedder that has already resolved the graph points this at its own file |
| `NIM_PARAMS` | (`statusgo.nims`) appended last to the nim-sds compiles (the channel nim-sds's tasks read) |
| `STATUS_GO_BUILD_DIR` | (Makefile) root of every library-target output; `STATUS_GO_BIN_DIR`, `STATUS_GO_BINDINGS_PATH`, `STATUS_GO_LIBRARY_OUT` and `STATUS_GO_STUB_BINDINGS_OUT` derive from it. Default: `./build` |

Artifact layout under `STATUSGO_BUILD_DIR`:

```
.sds-build/build/libsds.*     the nim-sds artifacts (statusgo.nims)
.sds-build/library/libsds.h   nim-sds's API header, library/libsds.h (statusgo.nims)
```

and under `STATUS_GO_BUILD_DIR`:

```
bin/libstatus.*               the Go library and its generated header
bin/statusgo-lib/             the generated cbindings entry point
```

### Generated Go sources are built outside the tree

The generated sources are not committed. The LIBRARY targets
(`statusgo-library`, `statusgo-shared-library`, `statusgo-android-library`,
`statusgo-ios-library`) never generate into the tree either, so a checkout and
the store copy build the same way:

1. `make generate-overlay` runs `scripts/generate-overlay.sh`, which executes the
   `//go:generate` directives the library needs (`protoc`, `go-bindata`, the
   endpoint and handler tables) with their output redirected to
   `$(STATUS_GO_BUILD_DIR)/generated/src/<package path>`, and writes
   `$(STATUS_GO_BUILD_DIR)/generated/overlay.json`. It reruns only when a
   `.proto`, `.sql`, template or generator changed.
2. The targets pass `-overlay=.../overlay.json` to `go build`, which then sees
   those files as if they sat in their package directories.

This needs `protoc` on `PATH`; `protoc-gen-go` and `go-bindata` come from the
module (`go tool`). The directives are the single source: the script rewrites
only their output path and fails on a directive shape it does not know. Mocks
and contract bindings are not part of the library and are left to
`make generate`, which still generates in place for tests, linters and editors.

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

## Native dependency: libsds (nim-sds)

Every build links `libsds`, built from [nim-sds](https://github.com/logos-messaging/nim-sds)
by its own nimble tasks. Outside the Nix shell the only Nim-side prerequisite is
[nimble](https://github.com/nim-lang/nimble/releases) 0.24.1 on `PATH`: nim-sds
pins its compiler (`nim == 2.2.10`) and `nimble setup` materialises it into
nimble's store, so no `nim` needs to be installed (and one on `PATH` is not used
for this build).

- `make build-libsds` clones the pinned revision (`NIM_SDS_REPO`, `NIM_SDS_VERSION`
  in the Makefile) into `NIM_SDS_SOURCE_DIR` (default: `../nim-sds` next to this
  checkout), runs `nimble setup` there and then the host's `libsdsDynamic<OS>` task
  which writes to `NIM_SDS_LIB_DIR` (`<source dir>/build`). The header cgo
  compiles against is `library/libsds.h` in the nim-sds tree (`NIM_SDS_INC_DIR`).
- `make build-libsds-android ARCH=…` / `make build-libsds-ios` run the per-target
  tasks the same way (`libsdsAndroid<Arch>`, `libsdsIOS`).
- To link a prebuilt `libsds` instead, pass both `NIM_SDS_LIB_DIR` and
  `NIM_SDS_INC_DIR`; nothing is cloned or built then (this is what the Nix shell
  does).

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
