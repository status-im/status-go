mode = ScriptMode.Verbose

import std/[os, strutils]

### Package
version     = "0.0.1"
author      = "Status Research & Development GmbH"
description = "Status messaging and wallet backend, built as a C archive for status-app"
license     = "MPL-2.0"

# status-go is a Go project; this package exists to resolve its Nim native
# dependencies through Nimble, and to let status-app depend on status-go the
# same way (status-im/status-app#19907).
#
# srcDir points at an empty directory and every Go tree is skipped, so this
# contributes dependencies, not Nim modules. The Makefile orchestrates the
# build: Nimble resolves the Nim side, Go resolves the Go side, and Make links
# them together.
srcDir = "internal/nimble/src"


### Dependencies
requires "nim >= 2.2.4"

# The bindings pin the nim-sds revision their C ABI matches, and it resolves
# transitively. A pin there overrides anything named here, so this does not name
# one; that changes when nim-sds publishes versioned tags and the bindings can
# express a range instead.
requires "https://github.com/logos-messaging/sds-go-bindings#c94172d"

# Same arrangement for the delivery side: the bindings pin the liblogosdelivery
# revision their C ABI matches, and it resolves transitively.
requires "https://github.com/logos-messaging/logos-delivery-go-bindings#280a7889"


### Helpers

proc nimblePkgDir(name: string): string =
  ## Where the dependency was installed. `nimble path` prints one line per
  ## installed version and exits 0 even when the package is missing, so take the
  ## one that carries its nimble file rather than trusting position.
  let (output, _) = gorgeEx("nimble path " & name)
  for line in output.strip().splitLines():
    let candidate = line.strip()
    if candidate.isAbsolute() and fileExists(candidate / (name & ".nimble")):
      return candidate
  raise newException(CatchableError, name & " unresolved - run `nimble setup`")

### Tasks

proc runBindingsTask(taskName: string) =
  ## Delegates to the bindings' own task; they pin the nim-sds revision their
  ## C ABI matches. LIBSDS_OUT and NIM_PARAMS come from the caller.
  withDir nimblePkgDir("sds_go_bindings"):
    exec "nimble " & taskName

task libsds, "Build the libsds status-go links against":
  runBindingsTask("libsds")

task libsdsAndroid, "Build libsds for Android; ARCH selects the architecture":
  runBindingsTask("libsdsAndroid")

task libsdsIOS, "Build libsds for iOS":
  runBindingsTask("libsdsIOS")

task liblogosdelivery, "Build the liblogosdelivery status-go links against":
  ## LIBLOGOSDELIVERY_OUT and NIM_PARAMS come from the caller.
  withDir nimblePkgDir("logos_delivery_go_bindings"):
    exec "nimble liblogosdelivery"
