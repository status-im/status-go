# Go Project Layout (#7067) — Status Audit & Remaining Plan

Baseline: `develop` @ `c3df487e3` (2026-08-19).

The issue's checklist runs to number 38 but skips 2, 5, 34 and 37 — **34 actual items**.
Against `develop` today:

- **13 done** (12 clean, 1 landed somewhere other than where the issue said — item 22)
- **7 in review** — Wave 1, shipped as the stacked PRs #7723 → #7724 → #7725 → #7726
- **9 partially done** — the move landed, the "clean it up / delete it" half is open
- **5 not started**

Twelve PRs landed the done part: #7116, #7190, #7191, #7205, #7206, #7207, #7209, #7214,
#7222, #7223, #7224, #7226.

---

## 1. Item-by-item status

| # | Item | Status | Evidence / what's left |
|---|------|--------|------------------------|
| 1 | `accounts-management` cleanup + rename to `accounts` | 🟡 partial | Moved to `internal/accounts-management`. Still hyphenated, not renamed, no cleanup. Name collides with `services/accounts`. |
| 3 | `api` → `pkg` | 🟡 mostly | `api/` is gone → `pkg/backend`. See breakdown below. |
| 3a | Remove "signing phrase" | 🔴 | Still live: `pkg/backend/defaults.go:76,410` (`buildSigningPhrase`), `Settings.SigningPhrase`, `signing_phrase` DB column. |
| 3b | Remove `StatusBackend` interface | ✅ | No `type StatusBackend interface` anywhere. |
| 3c | `GethStatusBackend` → `StatusBackend` | ✅ | Only stale *test helper* names remain (`setupGethStatusBackend` in `pkg/backend/backend_test.go`). Trivial. |
| 3d | Goes to `pkg` | ✅ | `pkg/backend`. |
| 3e | `app_state` → non-app-related args | ✅ | No `app_state` / `AppStateChange` left. |
| 3f | `create_account_and_login` placement | ✅ | Lives with the backend (`pkg/backend`). |
| 3g | `default_networks` → `networks` package | 🟡 | Went to `params/networkdefaults`, not a networks service. Folded into #7247. |
| 3h | Split `Settings` per service | 🔴 | `internal/db/multiaccounts/settings` is still one monolithic struct. **Blocked on design decision D6.** |
| 3i | `NodeConfig` dies | 🔴 | `params.NodeConfig` still referenced in **108** files. **Blocked on D6.** |
| 3j | `multiformat` → `pkg/multiformat` | ✅ | Done. |
| 4 | Databases → `internal/db` | 🟡 mostly | `internal/db/{appdatabase,walletdatabase,multiaccounts,sqlite}` ✅. Open: rename `walletdatabase`→`walletdb` and split "migrations" vs "db file creation"; the "clean up multiaccounts A LOT" part is untouched. |
| 6 | `centralizedmetrics` → `services/` | 🟡 | Landed as `internal/metrics` instead. **Decision D4.** |
| 7 | `circuitbreaker` → `internal`, then delete | 🟡 | `internal/circuitbreaker` ✅. Deletion 🔴 (**D5**). |
| 8 | `cmd` cleanup | ✅ | `cmd/generate_handlers` and `cmd/library` both gone. `cmd/` = `status-backend`, `push-notification-server`. |
| 9 | `common/dbsetup` → `internal/db` | 🔵 in review | Moved to `internal/db/dbsetup` in #7724. |
| 10 | `common/device.go` → `internal/platform` | 🔵 in review | Now `internal/platform`, in #7724. |
| 11 | `internal/connection` | ✅ | Done. |
| 12 | `constants` split | 🔵 in review | Split three ways in #7724: archive/torrent paths + mainnet URL → `params`, IPFS gateway → `internal/ipfs.GatewayURL`. |
| 13 | `internal/contracts` | ✅ | Done. |
| 14 | `crypto` → `internal` (then out) | 🟡 | `internal/crypto` ✅. Removal 🔴 (**D5**). |
| 15 | `deprecation` dies | ✅ | Removed in #7209 (profile/timeline chats gone with it). |
| 16 | `errors` → `internal` | ✅ | `internal/errors`. |
| 17 | `eth-node` out | ✅ | Gone. |
| 18 | `healthmanager` → `internal` | ✅ | Done. |
| 19 | `images` → `internal` | ✅ | Done. |
| 20 | `ipfs` → `internal` | 🟡 | Move ✅. "Disable in privacy mode / run own node" is a **product feature**, not layout — should leave this issue. |
| 21 | `logutils` → `internal` | ✅ | Done. |
| 22 | `messaging` → `internal` | 🟢 diverged | Landed as **`pkg/messaging`** (public), not internal. Almost certainly right — confirm and amend the issue (**D3**). |
| 23 | `mobile` → split into services | 🔴 | `mobile/status.go` is **1941 LOC**. Tracked separately in **#7079**. |
| 24 | `node` → `pkg/backend` | ✅ | `pkg/backend/node`. |
| 25 | `params/networkhelper` → `services/networks` | 🔴 | `params/networkhelper` + `params/networkdefaults` still under `params/`. Tracked in **#7247**. |
| 26 | `cluster` → out / split | 🔴 | `params/cluster.go` (103 LOC) still there. **D5.** |
| 27 | `protocol` → `internal` | 🔴 | **392 Go files, 331 importing files.** Not started. |
| 28 | `rpc` → `internal` | ✅ | `internal/rpc`. |
| 29 | `localpairing` → service | 🔵 in review | Now `services/pairing`, in #7725. |
| 30 | `server_media` → `services/media` | 🔵 in review | Now `services/media`, in #7725; generic HTTP plumbing → `internal/httpserver`. |
| 31 | `services` → `pkg` | 🔴 | **545 Go files, 375 importing files.** Not started. |
| 32 | `signals` → `internal`, non-global | 🔵 in review | Moved to `internal/signal` in #7723. The "no global functions" half is left out — behavioural, touches the C-binding callback. |
| 33 | `static` split + `go:embed` instead of bindata | 🟡 | Asset bindata is gone ✅. **SQL migrations still use `go-bindata`** — 13 generator dirs: 4 under `internal/db/`, 3 under `protocol/`, 5 under `pkg/messaging/`, 1 under `services/newsfeed/`. |
| 35 | `t` → `internal` | ✅ | → `internal/testutils`. |
| 36 | `tests-functional` + `tests-unit-network` → `/test` | 🔵 in review | Now `test/functional` and `test/unit-network`, in #7726. |
| 38 | `transactions` → `internal`, then out | 🟡 | `internal/transactions` ✅. Removal + `MessageSigner` interface 🔴 (**D5**). |

### What's still at the repo root

```
common/  (12 files)   mobile/ (11)    params/ (17)    protocol/ (392)
server/  (51)         services/ (545) signal/ (19)    tools/ (6)
pinned-communities/ (1)   cmd/ (7)    tests-functional/   tests-unit-network/
```

`common/` is by far the cheapest kill: **~500 non-test LOC across 12 files**, and it is
explicitly called out in the issue preamble as the kind of package that must not exist.

---

## 2. Plan

Ordering principle: drain everything small and semantic **before** the two mega-moves
(`protocol/`, `services/`), because those are pure `git mv` + import rewrites that conflict
with every open branch in the repo. They should land in a single announced freeze window,
not trickle.

### Wave 1 — SHIPPED FOR REVIEW as a four-PR stack

| PR | Branch | Scope | Base | Files |
|---|---|---|---|---|
| [#7723](https://github.com/status-im/status-go/pull/7723) | `refactor/7067-signal-internal` | `signal/` → `internal/signal/` | `develop` | 80 |
| [#7724](https://github.com/status-im/status-go/pull/7724) | `refactor/7067-common` | dissolve `common/` | #7723 | 158 |
| [#7725](https://github.com/status-im/status-go/pull/7725) | `refactor/7067-server` | dissolve `server/` | #7724 | 87 |
| [#7726](https://github.com/status-im/status-go/pull/7726) | `refactor/7067-tests-dir` | tests → `test/` | #7725 | 202 |

Every commit in the stack passes `go vet ./...` across the whole tree **individually**, so the
stack is bisectable and each PR reviews on its own.

Items **E** (rename `internal/accounts-management`) and **F** (`go:embed` for SQL migrations) were
dropped by decision: E is churn without payoff right now, F is out of scope for this ticket.


**A. Dissolve `common/`** — one PR, ~138 import sites, no logic change.

| From | To | Note |
|---|---|---|
| `common/dbsetup/` | `internal/db/dbsetup/` | item 9 |
| `common/devices.go` | `internal/platform/platform.go` | item 10 |
| `common/pausable*.go` | `internal/pausable/` | genuinely cross-cutting (6 consumers: ipfs, rpclimiter, backend/node, messaging/transport, services/backup, protocol/ens) |
| `common/utils.go:LogOnPanic` | `internal/logutils/` | ⚠️ also update `Makefile:604` — the panic linter is pinned to `.../common.LogOnPanic` |
| `common/utils.go:RecoverKey`, `ValidateDisplayName` | `protocol/common/` | protocol-specific (imports `protocol/protobuf`) |
| `common/utils.go:IsNil`, `Ptr`, `TruncateWithDot*` | inline at consumers, or `internal/goutil` | **D7** |
| `common/errors.go:ErrBigIntSetFromString` | `services/wallet/` | wallet-only consumer |
| `common/status_node_service.go:StatusService` | `pkg/backend/node/` | that's the only consumer |
| `common/constants.go` | split 4 ways: archive paths → `protocol/communities`, `MainnetEthereumNetworkURL` → `params/networkdefaults`, `IpfsGatewayURL` → `internal/ipfs` | item 12 |

**B. Dissolve `server/`** — one or two PRs.

| From | To | Note |
|---|---|---|
| `server/pairing/` | `services/pairing/` | item 29 |
| `server/server_media.go`, `server_media_interface.go`, `handlers.go`, `handlers_linkpreview.go` | `services/media/` | item 30 |
| `server/{server.go,certs.go,ips.go,timeout.go,device.go,listen_control_*.go,servertest/}` | `internal/httpserver/` | the leftover generic HTTP/TLS plumbing both services need |

**C. `signal/` → `internal/signal/`** — pure move, 57 import sites. Keep the global API for
now; the "signals not via global functions" redesign is a separate behavioural change and
should be its own issue (it touches `mobile/` and the C-binding callback).

**D. Tests → `/test/`** — `tests-functional/` → `test/functional/`,
`tests-unit-network/` → `test/unit-network/`. Touches the Makefile, `tests-functional/Makefile`,
Dockerfiles, CI (Jenkins/GH workflows) and `tests-functional/README.MD`. Zero Go impact.

**E. Rename `internal/accounts-management` → `internal/accounts`** (item 1) — ~~DROPPED~~ — and resolve the
collision with `services/accounts` (the service is the RPC surface, the internal package is the
keystore/manager; the naming should say so, e.g. `internal/accounts` + `services/accounts`
is fine as-is once one of them is documented, or rename the internal one `internal/keystore`).

**F. `go:embed` for SQL migrations** (item 33 remainder) — ~~OUT OF SCOPE, needs its own issue~~ — replace `go tool go-bindata` in the
13 `.../migrations/**/doc.go` generators with `//go:embed`. Real benefit: `go build ./...`
currently **fails on a clean checkout** without `make generate`; `go:embed` removes that step
and unblocks plain-IDE workflows. Needs care: `status-im/migrate` expects a `bindata.AssetSource`,
so a small `embed.FS` → `AssetSource` shim is required (or switch to the `iofs` source driver).

### Wave 2 — needs a design decision first

**G. `params/` dissolution** (items 25, 26, 3g, 3i)
- `params/networkhelper` + `params/networkdefaults` → `services/networks` (#7247)
- `params/cluster.go` → delete/split (**D5**) — first verify no client reads it
- `params/config.go` (`NodeConfig`) → the per-service settings redesign (**D6**)

**H. Settings split + `NodeConfig` death** (items 3h, 3i) — the single biggest remaining item,
and the one the issue itself leaves unresolved (two competing designs). Everything in `params/`
and half of `mobile/` hangs off it. **This should be its own RFC-style issue with a decision,
not a bullet in #7067.**

**I. Deletion candidates** (items 7, 14, 26, 38) — `internal/circuitbreaker`, `internal/crypto`,
`internal/transactions`, `params/cluster.go`. Each needs "what replaces it" answered (**D5**).

### Wave 3 — the two mega-moves (freeze window)

**J. `services/` → `pkg/services/`** — 545 files, 375 import sites.
**K. `protocol/` → `internal/protocol/`** — 392 files, 331 import sites.

Both are `git mv` + a scripted import rewrite + `goimports -local github.com/status-im/status-go`.
Mechanically trivial, socially expensive. Recommendation: **one PR each, merged back-to-back on
an announced day**, after Wave 1 has drained and with no long-lived feature branch open
(check with the wallet team and whoever owns the Nimble-migration branch first — **D8**).

Do **not** interleave these with Wave 1: every Wave 1 PR would need a rebase.

### Wave 4 — already tracked elsewhere

- **#7079** — extract services out of `mobile/status.go` (1941 LOC). Item 23.
- **#7242** — move wallet-related services under `services/wallet`.
- **#7247** — move network functionality into a Networks service. Items 25, 3g.

These are real work but already have homes; #7067 should link and stop restating them.

### Suggested sub-issue split for #7067

The issue asks to be split. Proposed:

1. `chore: dissolve common/` (Wave 1A)
2. `chore: dissolve server/ into services/media + services/pairing` (1B)
3. `chore: move signal/ to internal/` (1C)
4. `chore: move tests to /test` (1D)
5. `chore: replace go-bindata migrations with go:embed` (1F)
6. `refactor: move services/ to pkg/services/` (3J)
7. `refactor: move protocol/ to internal/protocol/` (3K)
8. `design: per-service settings, retire NodeConfig` (2H) — decision issue
9. `chore: remove signing phrase` (3a)
10. `chore: delete circuitbreaker / crypto / transactions / cluster` (2I) — one per package

---

## 3. Decisions I need from you

| # | Question | My recommendation |
|---|---|---|
| **D1** | `protocol/` → `internal/protocol` makes it unimportable outside the module. Is anything outside status-go importing it as a Go package (status-cli, tests, other repos)? | If nothing external imports it → `internal/protocol`, as the issue says. If something does → `pkg/protocol`. I couldn't verify external consumers from inside this repo. |
| **D2** | `services` → `pkg`: flat (`pkg/wallet`, `pkg/ens`, …) or grouped (`pkg/services/wallet`)? | **Grouped** (`pkg/services/<name>`). 27 top-level dirs in `pkg/` would be worse than what we have now. |
| **D3** | Item 22 says `messaging` → `internal`; it actually landed in `pkg/messaging`. | Keep `pkg/messaging` — it's the reusable core and CLAUDE.md already documents it as such. Amend the issue text. |
| **D4** | Item 6 says `centralizedmetrics` → `services/`; it landed as `internal/metrics`. | Keep `internal/metrics` if it has no RPC surface. Move to `services/metrics` only if clients call it over RPC — you'll know which. |
| **D5** | The four "AFUERA" packages — `circuitbreaker`, `crypto`, `transactions`, `params/cluster`. What replaces each, and is deletion in scope for this issue or a follow-up? | Split each into its own issue with an owner; they're architectural removals, not moves, and they'll stall #7067 if bundled. |
| **D6** | Per-service settings: `<Service>Config` + `Set<Setting>` **vs** `type Settings interface` with `Get<Setting>` / `GetSetting <-chan T`. The issue records both, undecided. | Needs you + backend leads. This gates `NodeConfig` death, the settings split, `params/` dissolution and half of `mobile/`. Suggest making it a decision issue with a written proposal rather than leaving it in the checklist. |
| **D7** | ~~Where do the generic helpers go?~~ **Answered in #7724.** | `IsNil` had 2 call sites and `Ptr` 7 across two test files → inlined as unexported helpers. `TruncateWithDot` has 122 call sites, all inside log lines or error messages → `internal/logutils` as redaction. No junk-drawer package created. Say the word if you'd rather have `internal/goutil`. |
| **D8** | Timing for the two mega-moves (J, K). Who has long-lived branches that would be destroyed by a 700-file rename? | Pick a date, announce it, land both PRs the same day. I'd do it right after Wave 1 drains. |
| **D9** | Item 20's IPFS sub-points ("disabled in privacy mode", "use own IPFS node") are product features, not layout. | Move them out of #7067 into a product issue. |
| **D11** | `services/media` still exports `MediaServer`, so `media.MediaServer` stutters. Rename to `media.Server` / `media.Interface`? | ~40 call sites. My weak preference is a follow-up — a relocation PR that also renames types is harder to review. Say the word and I'll fold it into #7725. |
| **D12** | Splitting `server` forced a test-only method onto `httpserver.Server`. | The MediaServer URL tests set `address`, `cachedPort` and `config` directly. #7725 adds `ListeningAddr()` and `CachedPort()` (defensible public API) plus a marked `SetURLStateForTest`. Worth a look from whoever owns that code. |
| **D13** | The Nix dev shell cannot be built on `linux-arm64`. | `codecov-cli` 404s for the linux-arm64 platform, failing the whole shell derivation; and the pinned `status-im/go-sqlcipher` fork doesn't compile under GCC 14 on aarch64. Neither is caused by this work — both reproduce on untouched `develop` — but together nobody on ARM Linux can build or test status-go. Deserves its own issue. |
| **D10** | Item 4: `multiaccounts` "needs to be cleaned up A LOT" is unscoped. What does "cleaned up" mean concretely? | Needs your definition before anyone can estimate it. |

---

## 4. Operational notes for whoever executes this

- `go build ./...` **fails on a clean checkout** — `pkg/version` and `pkg/sentry` use `go:embed`
  on generated files, and 5 migration packages are bindata-generated. `make generate` first.
  Wave 1F (go:embed) plus generating the version/sentry stamps would fix this permanently and is
  worth doing early — it makes every subsequent move verifiable in a plain IDE.
- After every move: `make generate && make lint && make test`, plus a functional-test run —
  `tests-functional/` hard-codes nothing Go-path-shaped, but the Makefile paths matter.
- Import rewrites: `goimports -local github.com/status-im/status-go` is enforced by
  `.golangci.yml`, so run `make lint-fix` after any `git mv`.
- The panic linter is pinned to the symbol path in the Makefile itself:
  `go tool goroutine-defer-guard -target github.com/status-im/status-go/common.LogOnPanic ./...`
  (`Makefile:604`). Update that line in the same PR that moves `LogOnPanic`, or `make lint-panics`
  silently passes everything.
