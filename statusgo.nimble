# Package
version       = "0.1.0"
author        = "Status Research & Development GmbH"
description   = "status-go: the Status protocol node (Go); owns the pins and builds of the Nim libraries status-go links via cgo"
license       = "MPL-2.0"

# The package ships the `status_go` Nim wrapper (status_go.nim + status_go/)
# alongside the Go sources that produce libstatus, so wrapper and C API version
# together. Build tasks live in statusgo.nims — this manifest must stay
# declarative for nimble's parser. See docs/building.md, "status-go as a nimble
# package".

# SOURCE-ONLY manifest: no `bin`, no install whitelists, no build hook. On
# nimble 0.24.1 any of those makes every consumer pay a binary build inside
# `nimble setup`, or strips the store copy to the whitelisted files, dropping
# go.mod, the Makefile and the Go sources, which a consumer builds in place
# with the Makefile's library targets.

# nim-sds pin: the single source of truth for both revision and repository;
# the Makefile derives NIM_SDS_VERSION and NIM_SDS_REPO from the line below.
# This is nim-sds's master line: the snake_case JSON C ABI that the
# sds-go-bindings pseudo-version in go.mod links against (the bindings were
# bumped to it on develop), built by the package's own nimble tasks. The
# branch `nimble-embed` of alexjba/nim-sds is upstream master
# (logos-messaging/nim-sds 04441cb) plus only what embedding needs: every
# output under SDS_OUT_DIR, library/sds_tasks.nims as the task entry point
# without nimble, and the compiler pin aligned to nim 2.2.10, which is what
# lets statusgo.nims build a read-only store copy in place. Moves to the
# upstream tag once that branch merges there.
requires "https://github.com/alexjba/nim-sds.git#d0bd6f914123d7cf4a2f0fd7e9f133f6be1f04c3"
