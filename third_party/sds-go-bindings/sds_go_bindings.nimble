mode = ScriptMode.Verbose

import std/[os, strutils]

### Package
version     = "0.4.0"
author      = "Logos"
description = "Go bindings for nim-sds"
license     = "MIT or Apache License 2.0"

# This is a Go module. The Nimble package exists so consumers get a libsds
# whose C ABI matches these bindings, resolved and built the same way.
#
# Pinned to a commit: nim-sds has no v0.4 tag yet, so a range cannot express
# "at least the one whose installed package can build libsds". This becomes a
# range once it publishes one, and consumers can then upgrade without a release
# here.
#
# srcDir points at an empty directory and the Go tree is skipped, so nothing is
# contributed to a dependent's Nim path.
srcDir = "internal/nimble/src"

skipDirs = @["sds", "internal"]

### Dependencies
requires "nim >= 2.2.6"
# nimble.lock pins the untagged chronos commit nim-ffi needs; a pin here breaks dependents.
requires "chronos"
requires "https://github.com/logos-messaging/nim-sds#4b08d508"

### Helpers

proc nimblePkgDir(name: string): string =
  ## Where the dependency was installed. `nimble path` prints one line per
  ## installed version and exits 0 even when the package is missing.
  result = getEnv("SDS_PKG_DIR")
  if result.len > 0:
    if not dirExists(result / "library"):
      raise newException(CatchableError,
        "SDS_PKG_DIR=" & result & " has no library/")
    return

  let (output, exitCode) = gorgeEx("nimble path " & name)
  for line in output.strip().splitLines():
    let candidate = line.strip()
    if candidate.isAbsolute() and dirExists(candidate / "library"):
      return candidate

  raise newException(CatchableError,
    name & " unresolved. `nimble path " & name & "` exited " & $exitCode &
    " and returned:\n" & output.strip() &
    "\nSet SDS_PKG_DIR to the package directory to bypass this lookup.")

### Tasks

# nim-sds names every platform's output libsds.*; DynlibFormat would drop the
# lib prefix on Windows.
const hostLib = "libsds." & (when defined(windows): "dll"
                             elif defined(macosx): "dylib"
                             else: "so")

proc runNimSdsTask(taskName, lib: string) =
  ## Delegates to nim-sds' own build task, then copies `lib` and the header to
  ## LIBSDS_OUT. Consumers set NIM_PARAMS (their resolved --path set) and
  ## LIBSDS_OUT; neither is decided here.
  let pkgDir = nimblePkgDir("sds")
  withDir pkgDir:
    exec "nimble " & taskName

  let outDir = getEnv("LIBSDS_OUT")
  if outDir.len > 0:
    mkDir outDir
    cpFile pkgDir / "build" / lib, outDir / lib
    cpFile pkgDir / "library" / "libsds.h", outDir / "libsds.h"

task libsds, "Build the libsds these bindings link against":
  runNimSdsTask("libsds", hostLib)

task libsdsAndroid, "Build libsds for Android; ARCH selects the architecture":
  runNimSdsTask("libsdsAndroid", "libsds.so")

task libsdsIOS, "Build libsds for iOS":
  runNimSdsTask("libsdsIOS", "libsds.a")
