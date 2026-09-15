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
# nimble 0.22.3 any of those makes every consumer pay a binary build inside
# `nimble setup`, or strips the store copy to the whitelisted files, dropping
# go.mod, the Makefile and the Go sources. status_backend is built from a
# checkout instead: `nim libstatus statusgo.nims` + `nim c status_backend.nim`.

# nim-sds pin (v0.3.3): the single source of truth for both revision and
# repository — the Makefile derives NIM_SDS_VERSION and NIM_SDS_REPO from the
# line below. It must stay on the v0.3 line, which carries the SDS
# retrieval-hint provider on the CamelCase FFI ABI that the sds-go-bindings pin
# in go.mod links against; master/release-v0.4 moved to a snake_case CBOR ABI.
# The fork branch `nimble-v0.3.3` (alexjba/nim-sds) adds the nimble packaging
# that writes every build output under SDS_OUT_DIR, which is what lets
# statusgo.nims build a read-only store copy in place; the library ABI is
# untouched. Moves to an upstream tag once release/v0.3 carries those commits.
requires "https://github.com/alexjba/nim-sds.git#da16e36c2aeb0c53d94833b9daf6850972101982"
