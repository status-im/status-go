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

# Executable delivery: `nimble install` puts the status-backend RPC server on
# PATH as `status_backend`, built from source for the host. The before-build
# hook produces the Go artifacts (build/bin/libstatus.a + libsds.a) that the
# bin links through the wrapper's auto-link — the same path library consumers
# use — and installDirs carries them into the package store so installed
# copies keep linking.
bin         = @["status_backend"]
# A package with `bin` is treated as binary-only: nimble installs nothing but
# the binary plus these whitelists. The wrapper module and the hook-produced
# build/ artifacts must be listed explicitly or consumers cannot
# `import status_go` from the installed copy (hybrid behavior).
installDirs  = @["build", "status_go"]
installFiles = @["status_go.nim", "statusgo.nims"]

before build:
  # Building the status_backend bin needs the Go artifacts at link time.
  # STATUS_GO_SKIP_GO_BUILD=1 skips the Go build for source-only workflows:
  # existing build/bin artifacts are reused when present, otherwise the
  # binary step is cancelled.
  if getEnv("STATUS_GO_SKIP_GO_BUILD").len > 0:
    if fileExists("build/bin/libstatus.a"):
      echo "STATUS_GO_SKIP_GO_BUILD set: skipping the Go build, " &
        "linking status_backend against the existing build/bin artifacts"
    else:
      echo "STATUS_GO_SKIP_GO_BUILD set and no prebuilt build/bin/libstatus.a: " &
        "skipping the status_backend binary step (source-only, no Go build)"
      return false
  else:
    if findExe("go").len == 0:
      quit "status-go builds libstatus from source and needs the Go toolchain, " &
        "but `go` was not found on PATH. Install Go (https://go.dev/dl/) or set " &
        "STATUS_GO_SKIP_GO_BUILD=1 for a source-only install without the binary."
    if findExe("make").len == 0:
      quit "status-go builds libstatus via its Makefile, but `make` was not " &
        "found on PATH. Install make (Xcode CLT / build-essential / msys2) or " &
        "set STATUS_GO_SKIP_GO_BUILD=1 for a source-only install without the binary."
    exec "nim libstatus statusgo.nims"
# Interim fork pin: PR logos-messaging/nim-sds#85 head (the whole 6-patch
# queue: ffi pin, NIMFLAGS forwarding, libsdsStaticMac localization,
# installDirs whitelist, ZERO_AR_DATE, -fno-common). Moves to the
# logos-messaging merge SHA when PR #85 lands. No explicit nim-ffi pin
# needed: this sds revision pins ffi itself (#fb25f069, the CI-certified
# 0.1.4).
requires "https://github.com/alexjba/nim-sds.git#5c89d61f897b44b75f2f28978f9928960181cf95"
