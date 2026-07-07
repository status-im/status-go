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
# `nim libsds statusgo.nims` (shared libsds only).

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

# Interim fork pin: PR logos-messaging/nim-sds#85 head (the whole 6-patch
# queue: ffi pin, NIMFLAGS forwarding, libsdsStaticMac localization,
# installDirs whitelist, ZERO_AR_DATE, -fno-common). Moves to the
# logos-messaging merge SHA when PR #85 lands. No explicit nim-ffi pin
# needed: this sds revision pins ffi itself (#fb25f069, the CI-certified
# 0.1.4).
requires "https://github.com/alexjba/nim-sds.git#5c89d61f897b44b75f2f28978f9928960181cf95"
