# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

`status-go` is the Go backend library powering the [Status app](https://github.com/status-im/status-app)
(desktop & mobile) — its primary and main client. It bundles a messaging stack, an embedded Ethereum/wallet
layer, and the Status app business logic. The app links it as a **C-binding library** (built from `mobile/`;
despite the directory name, gomobile is no longer used — it's historical naming). Module path:
`github.com/status-im/status-go` (Go 1.26, CGO-heavy).

The API is exposed to the app only through these C-bindings (plus a separate HTTP server for media). The
`cmd/status-backend` HTTP server exposes the same API over JSON, but exists primarily to drive the
**functional tests** — the app itself does not use it.

Messaging transport is **logos-delivery** (the protocol formerly known as Waku) — much of the code still uses
`waku` naming. logos-delivery is used purely as the transport layer; status-go is not otherwise tied to Logos.
The messaging specifications Status implements are published at https://lip.logos.co.

> Layout: the repo is mid-migration toward the standard Go layout
> ([golang-standards/project-layout](https://github.com/golang-standards/project-layout), tracked in
> [status-im/status-go#7067](https://github.com/status-im/status-go/issues/7067)) — the backend now lives in
> `pkg/backend/` and the messaging core in `pkg/messaging/`, but much code (`protocol/`, `services/`,
> `mobile/`, …) still sits at the repo root, and some `docs/building.md` paths (e.g. `./api/`, `./messaging/`)
> are stale. Prefer the layout described below.

## Build

Builds require CGO and native libraries (`libsds` (nim-sds) for SDS reliability, optionally `logos-storage`), so
the Nix dev shell is strongly recommended — it provides the toolchain and pins the native dependencies.

```shell
make shell                 # enter Nix dev shell (or: nix develop --extra-experimental-features 'nix-command flakes')
make status-backend        # build ./build/bin/status-backend (status-go as an HTTP server)
make statusgo-library      # static C library for the current platform
make statusgo-shared-library
make statusgo-android-library / make statusgo-ios-library
```

To work in an IDE / build as a plain Go project, you must first generate sources:

```shell
make status-go-deps        # install required Go tools
make generate              # protobufs, SQL migration bindata, mocks (via go-generate-fast)
```

Run the server: `./build/bin/status-backend --address=localhost:12345` (full JSON API on that port; see
`cmd/status-backend/README.md` and `cmd/status-backend/API_REFERENCE.md`).

## Lint

```shell
make lint        # runs `make generate`, lint-panics, then golangci-lint
make lint-fix    # golangci-lint --fix
```

`lint-panics` enforces that goroutines defer `panics.LogOnPanic` (via the `goroutine-defer-guard` tool).
Enabled linters: errcheck, gosec, govet, ineffassign, misspell, unconvert, plus goimports formatting with
`github.com/status-im/status-go` as the local-import prefix (see `.golangci.yml`).

## Test

Testing has two pillars: Go unit/integration tests, and a large **Python functional-test suite**
(`test/functional/`) that exercises the full backend end-to-end. Both are first-class — many behaviors
(messaging delivery, communities, wallet flows) are primarily covered by the functional tests, so changes
to backend behavior usually need attention in both.

```shell
make test                  # alias for test-unit (short dev run); runs `make generate` first
make test-unit-race        # with -race
make test-unit-network     # tests needing real network access (./test/unit-network/...)
```

Tests use [testify/suite](https://godoc.org/github.com/stretchr/testify/suite). To run a single test, target
its package and use `-testify.m` (the suite is launched by its `TestXxxSuite` runner function, and `-testify.m`
selects the method within it):

```shell
make test-single PKG=./pkg/messaging/controller/processor TEST=^TestSDSWrappedMessages$
# or directly:
go test -v ./server/pairing -test.run TestSyncDeviceSuite -testify.m ^TestTransferringKeystoreFiles$
```

### Functional tests

End-to-end tests live in `test/functional/` and are written in **Python/pytest** (not Go). They spin up
`status-go` Docker containers and require local Waku + Anvil fleets. `make test-functional` runs the full
suite via Docker; for local iteration set up a venv in `test/functional/.venv` and run `pytest -m rpc`
(use `-k <name>` for a single test). See `test/functional/README.MD` for the full workflow, fixtures
(`backend_factory`, `backend_new_profile`, `backend_recovered_profile`), and the Apple-Silicon Rosetta caveat.

## Database migrations

SQL migrations are embedded via generated bindata. After adding a `.sql` file, regenerate with the matching
target (each maintains its own migration dir):

```shell
make migration            # app DB    -> internal/db/appdatabase/migrations/sql
make migration-wallet     # wallet DB -> internal/db/walletdatabase/migrations/sql
make migration-protocol   # protocol  -> protocol/migrations/sqlite
make migration-check      # verify migrations are consistent
```

## Commits & PRs

- Conventional Commits are enforced (`make commit-check`). Work off and PR into `develop`.
- CI requires ≥50% patch coverage. High-risk features should sit behind feature flags
  (`protocol/common/feature_flags.go` or `pkg/featureflags/`).

## Architecture

Request flow, outermost to innermost: **client app → `mobile` (C-bindings) → `pkg/backend` (StatusBackend) →
`pkg/backend/node` (StatusNode + RPC services) → `protocol` (Messenger) + `pkg/messaging` (logos-delivery
transport)**. Asynchronous results flow back to clients through the `signal` package.

- **`cmd/status-backend/`** — a standalone HTTP server exposing the full `mobile` API over JSON
  (`main.go` → `server.RegisterMobileAPI`). Not a mobile component; it exists mainly to drive the Python
  functional tests. The other binary is `cmd/push-notification-server`.

- **`mobile/`** — the CGO-exported public API surface (package `statusgo`, mainly `status.go`). Every
  exported function here is callable from the client app via C-bindings. This is the entry point for almost
  all app operations. (Directory name is historical — gomobile is no longer used.)

- **`pkg/backend/`** — `StatusBackend` (`geth_backend.go`) is the central orchestrator: owns the app/wallet
  SQLite DBs, multiaccounts, node lifecycle, accounts manager, and wires everything together. `pkg/backend/node/`
  holds `StatusNode`, which constructs and registers the RPC services.

- **`protocol/`** — Status messaging *application* logic. The hub is `Messenger` (`messenger.go`,
  `messenger_*.go`, `messenger_handler.go`). Subpackages: `communities/`, `contacts/`, `encryption`/`v1`,
  `protobuf/` (wire format), `requests/` (typed API request structs), `migrations/`, push notification
  client/server.

- **`pkg/messaging/`** — the layered messaging *core* sitting beneath the protocol. `Core` (`core.go`)
  composes pipeline layers under `layers/` (`transport`, `encryption`, `segmentation`, `reliability`) over
  `waku/` (the logos-delivery / go-waku transport integration, `gowaku.go`), driven by `controller/` (receive)
  and `controller/sender/` (send). See `pkg/messaging/waku/README.md` and `layers/encryption/README.md`.

- **`services/`** — JSON-RPC services registered on the node and exposed over RPC: `wallet/` (largest;
  balances, transactions, collectibles, swaps), `accounts/`, `ext/` & `wakuv2ext/` (messenger RPC bridge),
  `ens/`, `stickers/`, `communitytokens/`, `connector/` (dApp connector), `local-notifications/`,
  `logosstorage/`, etc.

- **`params/`** — `NodeConfig` and network/fleet/cluster configuration (`config.go`, `cluster.go`,
  `network_config.go`).

- **`signal/`** — typed async events ("signals") emitted to the host app (`events_*.go`); how the backend
  pushes messages, wallet updates, node state, etc. back to clients.

- **`internal/`** — internal-only libraries: `db/` (DB setup + migrations), `rpc/`, `crypto/`, `logutils/`
  (zap logging; use `logutils.ZapLogger()`), `connection`, `healthmanager`, `circuitbreaker`, `contracts/`
  (generated, lint-excluded).


## Conventions

- Native dependencies: builds link against `libsds` (nim-sds) and optionally `logos-storage` (nim). The
  `make` targets clone/build these; the `USE_LOGOS_STORAGE` toggle gates the storage path
  (`make storage-help` for details). Outside the Nix shell, missing C deps are the usual cause of build failures.
- Generated files (protobuf, migration bindata, mocks) are committed — run `make generate` after changing
  `.proto`, `//go:generate` directives, or SQL migrations rather than editing generated output.
- Every spawned goroutine must `defer panics.LogOnPanic()` (enforced by `make lint-panics`).
