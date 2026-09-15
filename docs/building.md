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
  with `SDS_OUT_DIR` set to `NIM_SDS_LIB_DIR` (`<source dir>/build`). The header cgo
  compiles against is `library/libsds.h` in the nim-sds tree (`NIM_SDS_INC_DIR`).
- `make build-libsds-android ARCH=…` / `make build-libsds-ios` run the per-target
  tasks the same way (`libsdsAndroid<Arch>`, `libsdsIOS`).
- To link a prebuilt `libsds` instead, pass both `NIM_SDS_LIB_DIR` and
  `NIM_SDS_INC_DIR`; nothing is cloned or built then (this is what the Nix shell
  does).

## Build values come from `-ldflags`

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
