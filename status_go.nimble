mode = ScriptMode.Verbose

import std/[os, strutils]

### Package
version     = "0.0.1"
author      = "Status Research & Development GmbH"
description = "Status messaging and wallet backend, built as a C archive for status-app"
license     = "MPL-2.0"

# status-go has no Nim code: this package only resolves its Nim dependencies.
srcDir = "internal/nimble/src"


### Dependencies
# Any newer Nim is fine; the bindings' lock picks the one that builds libsds.
requires "nim >= 2.2.6"

# Pins the nim-sds revision whose C ABI the bindings match.
requires "https://github.com/logos-messaging/sds-go-bindings#7896913b"

# The untagged chronos commit nim-ffi needs; the libsds compile uses this tree.
requires "chronos#0de7b335d0ad5557ad5ba71a4b7662f7b201750e"

# Pins the logos-delivery revision whose C ABI the bindings match.
requires "https://github.com/logos-messaging/logos-delivery-go-bindings#743010ac"


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

proc runDeliveryBindingsTask(taskName: string) =
  ## LIBLOGOSDELIVERY_OUT and NIM_PARAMS come from the caller.
  withDir nimblePkgDir("logos_delivery_go_bindings"):
    exec "nimble " & taskName

task liblogosdelivery, "Build the liblogosdelivery status-go links against":
  runDeliveryBindingsTask("liblogosdelivery")

task liblogosdeliveryAndroid, "Build liblogosdelivery for Android; CPU and ABIDIR select the architecture":
  runDeliveryBindingsTask("liblogosdeliveryAndroid")

task liblogosdeliveryIOS, "Build liblogosdelivery for iOS; IOS_SDK, IOS_ARCH and IOS_SDK_PATH select the target":
  runDeliveryBindingsTask("liblogosdeliveryIOS")
