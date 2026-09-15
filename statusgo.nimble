# Package
version       = "0.1.0"
author        = "Status Research & Development GmbH"
description   = "status-go: the Status protocol node (Go); owns the pins and builds of the Nim libraries status-go links via cgo"
license       = "MPL-2.0"

# The package ships the `status_go` Nim wrapper (status_go.nim + status_go/)
# alongside the Go sources that produce libstatus, so wrapper and C API always
# version together. nim-sds provides libsds (SDS reliability layer), linked by
# status-go via cgo. Build tasks live in statusgo.nims (the .nimble must stay
# declarative for nimble's parser); run `nimble setup` once, then
# `nim libstatus statusgo.nims` (host auto-link artifacts) or
# `nim libsds statusgo.nims` (shared libsds only). Those tasks write NOTHING
# into this tree: a consumer builds the read-only store copy in place and
# points STATUSGO_BUILD_DIR at its own output directory (see statusgo.nims).

# SOURCE-ONLY on this interim branch (issue 0010): no `bin`, no install
# whitelists, no build hook. On nimble 0.22.3 a dependency manifest with `bin`
# forces every store materialization through buildtemp + the before-build hook
# + a binary build (vnext.nim gates on `bin.len > 0` alone, hook-cancel aborts
# the whole setup), i.e. consumers would pay a full host Go build inside every
# clean-store `nimble setup` for artifacts they never link. Any whitelist
# entry (installDirs/installFiles/installExt) additionally strips the store
# copy to the whitelisted files — dropping go.mod/Makefile/*.go — so all of
# them must stay absent for the full source tree to materialize (see
# AGENTS.md, "nimble 0.22.3 resolution walls"). The status_backend RPC server
# remains buildable from a checkout via `nim libstatus statusgo.nims` +
# `nim c status_backend.nim`; the former `nimble install` bin contract returns
# when nimble can materialize dependencies without building their binaries.

# nim-sds pin: v0.3.3 — the tag status-go's own Makefile tracked before this
# manifest became the single source of truth (NIM_SDS_VERSION derives from
# this line). v0.3.3 lives on nim-sds release/v0.3: it carries the SDS
# retrieval-hint provider required by the sds-go-bindings pin in go.mod, on
# the CamelCase FFI ABI. master/release-v0.4 moved to the snake_case CBOR ABI,
# which these bindings do not link against, so the pin must stay on the v0.3
# line. Consumed from the fork branch `nimble-v0.3.3` (alexjba/nim-sds):
# v0.3.3 + three commits that make the release/v0.3 tree a nimble package (one
# root manifest, library/ kept in store copies, NIMFLAGS forwarding, PR #85's
# localized/reproducible static archives) and keep every build output out of
# its source tree (SDS_OUT_DIR + a committed sds.nims), so statusgo.nims can
# build the read-only store copy IN PLACE — the library ABI is untouched.
# Moves to an upstream tag when release/v0.3 carries those commits.
requires "https://github.com/alexjba/nim-sds.git#0f8dc8689b228910480ee7402248df254ddee2cd"
