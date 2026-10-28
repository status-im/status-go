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
requires "https://github.com/logos-messaging/sds-go-bindings#25831865"

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
  ## C ABI matches.
  withDir nimblePkgDir(pkgName):
    exec nimbleExe() & " " & taskName

proc buildLibsds(taskName: string) =
  ## Skipped when LIBSDS_OUT is unset: libsds is prebuilt.
  if getEnv("LIBSDS_OUT").len > 0:
    runBindingsTask("sds_go_bindings", taskName)

proc buildLiblogosdelivery(taskName: string) =
  ## Builds liblogosdelivery without RLN. Run last: it extends NIM_PARAMS.
  ## Skipped when LIBLOGOSDELIVERY_OUT is unset: liblogosdelivery is prebuilt.
  if getEnv("LIBLOGOSDELIVERY_OUT").len == 0:
    return
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

task libs, "Build libsds into LIBSDS_OUT and liblogosdelivery into LIBLOGOSDELIVERY_OUT":
  buildLibsds("libsds")
  buildLiblogosdelivery("liblogosdelivery")

task libsAndroid, "Build the libraries for Android; ARCH, ANDROID_NDK_ROOT and ANDROID_API select the target":
  let arch = getEnv("ARCH", "arm64")
  let (cpu, abiDir) = case arch
    of "arm64": ("arm64", "arm64-v8a")
    of "arm": ("arm", "armeabi-v7a")
    of "amd64", "x86_64": ("amd64", "x86_64")
    else: raise newException(CatchableError, "unsupported ARCH: " & arch)
  putEnv("ARCH", cpu)
  buildLibsds("libsdsAndroid")
  putEnv("CPU", cpu)
  putEnv("ABIDIR", abiDir)
  if getEnv("ANDROID_API").len > 0:
    putEnv("ANDROID_TARGET", getEnv("ANDROID_API"))
  buildLiblogosdelivery("liblogosdeliveryAndroid")

task libsIOS, "Build the libraries for iOS; ARCH, IPHONE_SDK and IOS_TARGET select the target":
  let arch = getEnv("ARCH", "arm64")
  let sdk = getEnv("IPHONE_SDK", "iphoneos")
  putEnv("IOS_SDK_PATH", gorge("xcrun --sdk " & sdk & " --show-sdk-path").strip())
  putEnv("ARCH", if arch == "x86_64": "amd64" else: arch)
  buildLibsds("libsdsIOS")
  putEnv("IOS_SDK", sdk)
  putEnv("IOS_ARCH", arch)
  if getEnv("IOS_TARGET").len > 0:
    putEnv("IOS_DEPLOYMENT_TARGET", getEnv("IOS_TARGET"))
  buildLiblogosdelivery("liblogosdeliveryIOS")
