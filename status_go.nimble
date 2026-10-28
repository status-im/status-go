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
requires "https://github.com/logos-messaging/sds-go-bindings#4a098383"

# The untagged chronos commit nim-ffi needs; the libsds compile uses this tree.
requires "chronos#0de7b335d0ad5557ad5ba71a4b7662f7b201750e"

# Pins the logos-delivery revision whose C ABI the bindings match.
requires "https://github.com/logos-messaging/logos-delivery-go-bindings#09d0df35"

# Untagged: the commit logos-delivery locks.
requires "nat_traversal#860e18c37667b5dd005b94c63264560c35d88004"


### Helpers

# Windows `exec` prefers nim's bundled nimble.exe: https://github.com/status-im/status-go/issues/7941
proc nimbleExe(): string = findExe("nimble").quoteShell

proc nimblePkgDir(name: string): string =
  ## Where the dependency was installed. `nimble path` prints one line per
  ## installed version and exits 0 even when the package is missing, so take the
  ## one that carries its nimble file rather than trusting position.
  let (output, _) = gorgeEx(nimbleExe() & " path " & name)
  for line in output.strip().splitLines():
    let candidate = line.strip()
    if candidate.isAbsolute() and fileExists(candidate / (name & ".nimble")):
      return candidate
  raise newException(CatchableError, name & " unresolved - run `nimble setup`")

### Tasks

proc runBindingsTask(pkgName, taskName: string) =
  ## Delegates to the bindings' own task; they pin the library revision their
  ## C ABI matches. The output dir and NIM_PARAMS come from the caller.
  withDir nimblePkgDir(pkgName):
    exec nimbleExe() & " " & taskName

task libsds, "Build the libsds status-go links against":
  runBindingsTask("sds_go_bindings", "libsds")

task libsdsAndroid, "Build libsds for Android; ARCH selects the architecture":
  runBindingsTask("sds_go_bindings", "libsdsAndroid")

task libsdsIOS, "Build libsds for iOS":
  runBindingsTask("sds_go_bindings", "libsdsIOS")

proc runDeliveryBindingsTask(taskName: string) =
  var params = getEnv("NIM_PARAMS") & " -d:disable_rln"
  when defined(windows):
    # Until logos-storage/nim-leopard#29: CMake picks MSVC from vcvars.
    putEnv("CC", "gcc")
    putEnv("CXX", "g++")
    putEnv("CMAKE_GENERATOR", "MSYS Makefiles")
    # Portable like logos-delivery's config.nims: no -march=native, SSSE3 baseline.
    params.add " -d:\"LeopardCmakeFlags=-DCMAKE_BUILD_TYPE=Release" &
      " -DCOMPILER_SUPPORTS_MARCH_NATIVE=FALSE -DCMAKE_CXX_FLAGS=-mssse3\""
    # Static libgomp needs dlfcn-win32's -ldl after it.
    params.add " -d:LeopardExtraLinkerFlags=-Wl,--push-state,-Bstatic,-lgomp,-ldl,--pop-state"
  putEnv("NIM_PARAMS", params)
  runBindingsTask("logos_delivery_go_bindings", taskName)

task liblogosdelivery, "Build the liblogosdelivery status-go links against":
  runDeliveryBindingsTask("liblogosdelivery")

task liblogosdeliveryAndroid, "Build liblogosdelivery for Android; CPU and ABIDIR select the architecture":
  runDeliveryBindingsTask("liblogosdeliveryAndroid")

task liblogosdeliveryIOS, "Build liblogosdelivery for iOS; IOS_SDK, IOS_ARCH and IOS_SDK_PATH select the target":
  runDeliveryBindingsTask("liblogosdeliveryIOS")
