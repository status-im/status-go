# status-go as a nimble package — agent notes

status-go is a hybrid nimble package: the `status_go` Nim wrapper (root
`status_go.nim` + `status_go/impl.nim`) travels with the repo that produces
`libstatus`, and `statusgo.nimble` owns the nim-sds/nim-ffi pins. Patterns
below were each learned the hard way; keep them.

## Manifest / tasks split

- `statusgo.nimble` must stay **fully declarative** (no imports/procs/tasks —
  only hook blocks are safe). Code in the manifest degrades or hangs nimble
  0.22.x vNext dependency evaluation for every consumer. Tasks live in
  `statusgo.nims`.
- `nimble setup` without a lock pays a full SAT solve (~10 min: it downloads
  candidate versions for every loose range in the dependency tree). Locks do
  not propagate to consumers, so consumers of this package pay it once per
  fresh dependency dir.

## Auto-link (status_go wrapper)

- `import status_go` auto-links the artifacts `nim libstatus statusgo.nims`
  places in `build/bin` (static `libstatus.a` + `libsds.a`), plus the per-OS
  syslibs/frameworks the Go runtime needs. Opt-out: `-d:statusGoNoAutoLink`
  (status-desktop uses it — it links the shared flavor it builds itself).
- Link archives by **explicit path, not `-l`**: `build/bin` also holds
  `libstatus.dylib` when the shared flavor is built, and `-lstatus` silently
  prefers the dylib — breaking the static contract without an error.
- macOS frameworks for a Go c-archive: CoreFoundation, Security, IOKit, plus
  `-lresolv` (verified sufficient for `gowaku_no_rln` tags).

## Static Nim libraries consumed by Nim executables

- Any static Nim library (libsds) linked into a Nim executable **must have
  its non-API symbols localized** (`ld -r -exported_symbol '_Sds*'` +
  `-fvisibility=hidden -fno-common`), or the embedded Nim runtime collides
  with the consumer's own runtime. nim-sds does this for iOS and (since the
  local patch) `libsdsStaticMac`. Audit with:
  `nm -gU build/bin/libsds.a` → only `_Sds*` symbols.

## Installable bin (`status_backend`) — hook + hybrid mechanics

- A package with `bin` is **binary-only by default**: nimble installs the bin
  plus `installDirs`/`installFiles` and nothing else. The wrapper sources
  (`status_go.nim`, `status_go/`) must be whitelisted explicitly or installed
  copies cannot `import status_go` (hybrid contract silently broken).
- The `before build` hook needs the Go artifacts at link time, so it runs
  `nim libstatus statusgo.nims`. In install context there is no
  `nimble.paths`; `depPathFlags` then runs a nested `nimble setup` — fast,
  because the outer nimble already installed our deps into its store. A bare
  `--nimblePath:<store>` does NOT work for the inner libsds compile (lazy-path
  resolution proved unreliable); only exact `--path` entries do.
- Hook `return false` = nimble error "Pre-hook prevented further execution":
  usable to *cancel* the binary step (skip switch without artifacts), but the
  install then fails rather than completing source-only.
- `nimble install <local path>` (even via `file://…` for an uncommitted-free
  test) builds **in the source dir** and symlinks `$NIMBLE_DIR/bin` to it; a
  store copy under `pkgs2/` appears only for the `file://`/URL flavor. Use
  `file://` + a throwaway local commit to exercise the real staging→store
  path without publishing anything.
- New C exports that wrap packages other than `mobile/` (e.g. the
  status-backend server, which itself imports `mobile` — import cycle) cannot
  be picked up by `tools/generate-cbindings`' parser; append them verbatim in
  the generator like `Free` (see `statusBackendRunServer` in const.go).
- Nim bins linking libstatus need `-d:noSignalHandler` (the Go runtime owns
  signals); ship it via `<bin>.nim.cfg` so only that program gets it.
- Long-nimble-dir trap: the nimcache C-file name embeds the store path; a
  very long `NIMBLE_DIR` overflows macOS's 255-byte filename limit and the
  build dies with `cannot open '….nim.c'`. Keep test nimble dirs short.

## sds build engine (statusgo.nims)

- **One rule: every sds task builds the nimble-resolved nim-sds copy** (from
  `nimble.paths`) — no env override, no sibling convention, no cloning. A
  resolved path outside the store (develop link) is built **in place**
  (artifacts in `<checkout>/build`); a store copy (`…/pkgs2/sds-…`) is copied
  to `.sds-build/` at the package root (wiped and re-copied each run, never
  installed) and built there so the store stays pristine.
- The scratch copy gets `chmod -R u+w` (store trees can be read-only) and a
  **materialized `nimble.paths`** (this package's resolution minus sds
  entries): nim-sds's own `config.nims` includes a sibling `nimble.paths`, so
  the inner compile resolves deps even for sds revisions without NIMFLAGS
  forwarding.
- The unpatched upstream sds at the current pin cannot produce a working
  macOS host static build: no NIMFLAGS forwarding (in-workspace builds get
  poisoned by parent-dir configs — chronicles defines break the ffi compile)
  and no `libsdsStaticMac` localization (linking `status_backend` against the
  unlocalized archive fails with 17 duplicate Nim-runtime symbols; verified).
  Both are fixed by the local nim-sds patch queue (upstream-bound); until it
  merges and the pin bumps, host-static flows need the patched checkout.

## Byte-reproducible library outputs (the compare-before-copy contract)

Consumers (status-desktop ADR 0003) rebuild library dependents through
`cmp -s new old || cp` — an unchanged rebuild must therefore be
byte-identical, or every build relinks (and re-signs) the app. Three
nondeterminism sources were found and fixed (issue 0004); keep all three:

- `tools/generate-cbindings` iterates Go maps (packages, files, scope
  objects) — map order is randomized per process, so the generated
  `statusgo-lib/main.go` reshuffled its exports every run and the WHOLE
  c-archive changed (~60 MB of byte churn). All three loops now iterate
  sorted keys.
- Go's link-time build ID differs run-to-run even with identical inputs;
  the mobile library recipes pass `-ldflags=-buildid=` (the ID is unused in
  c-archive/c-shared artifacts).
- `ar` member headers embed real mtimes: Go's own archive writer for
  c-archive (repacked in the Makefile via `ZERO_AR_DATE=1 libtool -static
  -no_warning_for_no_symbols`) and nim-sds's `ar rcs` repack after the
  `ld -r` localization pass (prefixed `ZERO_AR_DATE=1` in sds.nimble).

Audit rebuild determinism with two runs + `cmp`; if archives differ only at
char ~32/33 of an ar line it's the header mtime, otherwise extract members
(`ar x`) and diff to find real content churn.

## status-desktop single-graph consumption (issue 0003)

- status-desktop requires this package in its ONE dependency graph
  (`nim_status_client.nimble`): interim via an ABSOLUTE `file://` requires
  (nimble rejects a develop-linked package whose manifest contains `file://`
  requires — see the wall below — and statusgo.nimble carries the interim
  file:// nim-sds pin). Final form once the sds pin flips back to a URL:
  `requires "statusgo"` + `nimble develop --add:vendor/status-go`
  (name-form requires DO accept develop links).
- The app's store lives OUT of tree (`~/.cache/status-desktop-nimbledeps`,
  `APP_NIMBLE_DIR` in the desktop Makefile): `nimble setup` builds dependency
  package binaries (dnsclient, via libp2p), and Nim's parent-dir config walk
  poisons in-tree builds with the app's `config.nims` (and, for nested git
  worktrees, with an enclosing checkout's — unfixable from inside the repo).
- `vendor/status-go/nimble.paths` (what the statusgo.nims sds engine reads)
  is a byte-for-byte COPY of the app's `nimble.paths`, derived by the desktop
  and mobile Makefiles — never a second `nimble setup`. Entries are absolute,
  so the copy is valid from any directory. The former per-status-go cache
  (`~/.cache/statusgo-nimbledeps`) and its ~11-minute second solve are gone.
- The app compiles the wrapper with `-d:statusGoNoAutoLink` (set in the app's
  `config.nims`): desktop links the shared libstatus/libsds it builds itself.
- App-side pin consequences of merging the graphs: json_rpc bumped to 0.6.1
  (0.6.0 caps websock `< 0.4.0`, libp2p 2.x needs websock `>= 0.4.0`).
  The graph resolves libp2p 2.0.0 / websock 0.4.0 / lsquic 0.5.4 / ffi 0.1.4
  — NOT libp2p 2.1.x; see the special-version pre-binding wall below.
- `nimble lock` at the app root diverges from what setup resolves: file://
  packages (statusgo, sds) are omitted, their transitive-only picks (libp2p,
  lsquic, boringssl, protobuf_serialization, npeg) are omitted too, and some
  shared entries are recorded at versions the resolution did not pick
  (websock 0.3.0 in the lock vs 0.4.0 resolved). With a WARM store the lock
  works anyway, but on a CLEAN store `solveLockFileDeps` takes the divergent
  entry as a hard constraint and the graph goes unsatisfiable — hand-fix the
  entry to the resolved version (version + vcsRevision + checksums.sha1,
  copied from the materialized store dir `pkgs2/<name>-<ver>-<sha1>`), the
  established lock-hand-fix pattern. Always verify a fresh lock by wiping the
  store and re-running `make nimble-deps`.

## nimble 0.22.3 resolution walls (verified in source + experiments)

- **`file://` requires are legal only at top level or inside packages reached
  via `file://`** (`developfile.nim` refuses to LOAD a develop-linked package
  whose manifest has one: "'file://' requires are only allowed in top level
  requires or requires opened from a file:// require"). Consequence: while
  statusgo.nimble carries the interim file:// nim-sds pin, statusgo itself
  cannot be develop-linked — consumers must use a file:// requires for it too
  (chains of file:// are fine).
- **URL + range requires evaluate EVERY tag's manifest**, and one candidate
  with an unresolvable dependency kills the whole solve, not the candidate:
  the reachability check (`nimblesat.getSolvedPackages`) hard-fails with
  "Dependency X not found in the graph" for any package name mentioned by any
  ENUMERATED candidate version. nim-ffi 0.2.0 (outside sds's `< 0.2.0` cap!)
  requires `cbor_serialization@0.3.0`, which doesn't land in the version
  table → the entire graph is unsatisfiable. Fix: pin URL deps by #hash
  (special versions bind directly and skip tag enumeration) — sds's ffi pin.
- **Special-version pre-binding poisons ranges package-wide**: if ANY
  enumerated candidate manifest requires `pkg#hash`, nimble pre-binds `pkg`
  to that special ("Multiple dependencies require different special versions
  ... using X, ignoring Y" — first one wins). libp2p 1.15.x manifests require
  `lsquic#6ae249c5` (manifest version 0.0.1), so in consumer graphs lsquic
  presents as 0.0.1 and libp2p 2.1.x's `lsquic >= 0.5.4` can never hold —
  the solver lands on libp2p ≤ 2.0.0. Counter-measures that do NOT work
  (all verified): raising the range floor (enumeration is unfiltered),
  pinning libp2p by URL#hash from a file://-chained manifest (does not
  resolve), a consumer-side top-level lsquic #hash pin (top-level specials
  don't reliably carry the semantic version needed to satisfy `>= X` ranges
  — see next bullet).
- **Top-level URL#hash pins satisfy transitive version RANGES only when
  nimble attached `speSemanticVersion`** (the manifest version of the pinned
  commit) to the special — `satisfiesConstraint` otherwise returns false for
  any non-`verAny` range. This enrichment is unreliable for root-manifest
  pins: a pin whose version must satisfy a transitive floor (e.g. metrics
  HEAD for libp2p 2.1.x's `metrics >= 0.2.2`) can make the graph UNSAT while
  the pin-less graph solves. Pins that only need to satisfy `any version`
  requires are safe (all of the app's classic pins).
- **Version tables are nondeterministic across days**: candidate listings
  come from `(url, range)`-keyed clones under `~/.nimble/pkgcache` that are
  never refreshed (a clone from July 2 hides tags cut July 3), and anonymous
  GitHub API rate-limiting (60/hr) silently degrades discovery. If a version
  that exists upstream is missing from a solve's "Available versions" table,
  delete the stale `~/.nimble/pkgcache/<pkg>_<range>` clones and re-run.
- `nimble setup` builds dependency package binaries (any dep with `bin`,
  e.g. dnsclient) in `<nimbleDir>/buildtemp`; keep the nimble dir OUT of any
  source tree or Nim's parent-dir config walk poisons those builds with
  enclosing `config.nims` files (app-relative link inputs on a dependency's
  link line). `--skipParentCfg` would be the exact fix but cannot be applied:
  it is CLI-only (configs can't set it), and nimble rejects it itself
  (`Unknown option: --skipParentCfg`) — there is no nim-flag passthrough to
  setup-time dependency builds. Upstream ask: nimble should pass
  `--skipParentCfg` to dependency builds by default.

- **Develop links cannot satisfy URL`#hash` requires**: the solved-package →
  candidate binding (`nimblesat.nim` `solveLocalPackages`) accepts a
  candidate only if the special version is in its `metadata.specialVersions`;
  only store copies (via `nimblemeta.json`) carry that, a develop checkout
  never does — `nimble develop --add` is silently ignored ("not a valid
  dependency → simply ignored" per the develop-workflow docs). Adding a
  `nimble.lock` (+ `nimble sync`, the documented develop workflow) does NOT
  fix it: setup still binds the store copy even when the develop checkout's
  git HEAD equals the locked vcsRevision (verified empirically). Only
  version-agnostic requirements (name-form like `requires "statusgo"`, per
  the 0001 harness) accept develop links.
- **Local-checkout substitution that DOES work**: point the requires at the
  checkout with an ABSOLUTE `file://` path
  (`requires "file:///abs/path/to/nim-sds"`) — nimble then resolves the
  checkout itself (link semantics: nimble.paths points into the checkout, so
  edits propagate and statusgo.nims builds it in place). A RELATIVE form
  (`file://../nim-sds`) and a bare remote URL both fail silently: the
  dependency's own path is dropped from nimble.paths while its transitive
  deps still resolve. Machine-local manifest edit only — restore the pinned
  URL before shipping.
- **`file://` deps and sibling URL#hash requires drop each other**: with
  both `requires "file:///…/nim-sds"` and
  `requires "https://…/nim-ffi#<hash>"` in one manifest, BOTH packages
  silently vanish from nimble.paths (deps of the file:// package still
  resolve). While sds is file://-substituted, the explicit ffi pin must be
  disabled and ffi resolves through sds's own capped range instead.
- nim-ffi 0.1.5+ refuses to compile without a signal-ownership define:
  every libsds flavor is embedded in a host process, so ALL sds tasks in
  statusgo.nims pass `-d:noSignalHandler` (the mobile tasks historically
  omitted it and only compiled because ffi 0.1.4 didn't enforce).
- **URL + version-range requires don't resolve at all** ("Dependency <url>
  not found in the graph"): URL requirements only work as `#hash`/`#branch`
  special versions on this nimble.
- Name-form requires (e.g. `requires "statusgo"`) DO accept develop links —
  that is why the 0001 consumer harness works — but they need the package in
  the nimble registry for non-develop consumers (nim-sds is not registered).
- `file://` URLs in `requires` take nimble's local-path branch: the `#rev`
  suffix is not parsed (treated as part of the directory). To install a
  pinned local revision, serve the repo's `.git` over loopback dumb HTTP
  (`git update-server-info` +
  `python3 -m http.server <port> --bind 127.0.0.1 --directory .git`) and pin
  `http://127.0.0.1:<port>/#<rev>`; dumb HTTP forces a full clone, which is
  what makes pinned revs work.
- **Install action and setup materialize DIFFERENT store contents** for the
  same package: `nimble install`-driven dependency installs hoist the srcDir
  contents to the store-entry root and drop everything else (sds loses
  `library/`), while `nimble setup`-driven materialization keeps the full
  repo tree — same `name-version-checksum` directory either way. Non-srcDir
  files a consumer needs (libsds.h, the FFI wrapper) survive only via an
  `installDirs` whitelist in the dependency's own manifest.
- **Hooks staged in buildtemp cannot run nested nimble project actions**:
  `thereIsNimbleFile` deliberately returns false under `<nimbleDir>/buildtemp`,
  so a `before build` hook cannot `nimble setup` in place during installs.
  statusgo.nims bootstraps instead by copying the manifest to a scratch dir
  under the system temp, running `nimble setup` there (same NIMBLE_DIR →
  same store), and copying the generated nimble.paths back (rewriting the
  root-package entry to the real package dir).

## Proving consumption from outside (the package boundary seam)

Reusable harness for consumer-facing changes (used for issue 0001; reuse for
install/import proofs):

```sh
mkdir consumer && cd consumer && git init   # develop needs VCS
# consumer.nimble: declarative, `requires "statusgo"`
nimble develop --add:<abs path to this repo>  # maps statusgo -> local checkout
nimble setup -l                                # resolves sds/ffi transitively
(cd <this repo> && nim libstatus statusgo.nims)  # host auto-link artifacts
nim c -r consumer.nim                          # import status_go + real call
nim c -d:statusGoNoAutoLink consumer.nim       # must FAIL to link
```

Run the consumer **outside** any Nim workspace tree: Nim's parent-dir config
walk otherwise poisons the build with the embedding repo's `config.nims`
(inside a workspace, add `--skipParentCfg:on` + the workspace's nimble.paths).
