mode = ScriptMode.Verbose

import std/[cmdline, os, strutils]

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

# Pins the nim-sds revision whose C ABI matches the Go bindings in go.mod.
requires "https://github.com/logos-messaging/nim-sds#c4dc3cdfb1d7ca6f043489d0d97f55b269e61644"

# The untagged chronos commit nim-ffi needs; the libsds compile uses this tree.
requires "chronos#0de7b335d0ad5557ad5ba71a4b7662f7b201750e"


### Tasks

proc nimParams(): string =
  result = getEnv("NIM_PARAMS")
  if result.len == 0 and fileExists("nimble.paths"):
    result = readFile("nimble.paths")
  result = result.replace("\r", " ").replace("\n", " ")

proc sdsSourceDir(): string =
  for flag in parseCmdLine(nimParams()):
    if flag.startsWith("--path:"):
      let candidate = flag["--path:".len .. ^1].strip(chars = {'"'})
      if fileExists(candidate / "sds.nimble") and
          fileExists(candidate / "library" / "libsds.nim"):
        return candidate
  raise newException(CatchableError,
    "Could not find resolved nim-sds in NIM_PARAMS or nimble.paths")

proc outputDir(): string =
  result = getEnv("LIBSDS_OUT")
  if result.len == 0:
    result = getCurrentDir() / "build"
  result = absolutePath(result)
  mkDir result

proc envOrDefault(name, fallback: string): string =
  result = getEnv(name)
  if result.len == 0:
    result = fallback

proc compilerCommand(outDir: string): string =
  "nim c --skipParentCfg --mm:orc -d:useMalloc --opt:size " &
    nimParams() & " -d:ignoreLeakCloseDispatcher -d:ffiOutputDir=" &
    quoteShell(outDir / "generated") & " --nimcache:" &
    quoteShell(outDir / "nimcache")

proc copyHeader(pkgDir, outDir: string) =
  cpFile pkgDir / "library" / "libsds.h", outDir / "libsds.h"

task libsds, "Build the libsds status-go links against":
  let pkgDir = sdsSourceDir()
  let outDir = outputDir()
  let lib = "libsds." & (when defined(windows): "dll"
                         elif defined(macosx): "dylib"
                         else: "so")
  exec compilerCommand(outDir) & " --app:lib --out:" &
    quoteShell(outDir / lib) & " " & quoteShell(pkgDir / "library" / "libsds.nim")
  copyHeader(pkgDir, outDir)

task libsdsAndroid, "Build libsds for Android; ARCH selects the architecture":
  let pkgDir = sdsSourceDir()
  let outDir = outputDir()
  let arch = envOrDefault("ARCH", "arm64")
  let triple =
    case arch
    of "arm64": "aarch64-linux-android"
    of "amd64": "x86_64-linux-android"
    of "i386": "i686-linux-android"
    of "arm": "armv7a-linux-androideabi"
    else: raise newException(CatchableError, "Unsupported Android architecture: " & arch)
  let ndk = getEnv("ANDROID_NDK_ROOT")
  if ndk.len == 0:
    raise newException(CatchableError, "ANDROID_NDK_ROOT is required")
  let host = when defined(macosx): "darwin-x86_64" else: "linux-x86_64"
  let compiler = ndk / "toolchains" / "llvm" / "prebuilt" / host / "bin" /
    (triple & envOrDefault("ANDROID_API", "28") & "-clang")
  if not fileExists(compiler):
    raise newException(CatchableError, "Android compiler not found: " & compiler)
  exec compilerCommand(outDir) & " --app:lib --os:android --cpu:" & arch &
    " --cc:clang --clang.exe:" & quoteShell(compiler) &
    " --clang.linkerexe:" & quoteShell(compiler) & " --out:" &
    quoteShell(outDir / "libsds.so") & " " & quoteShell(pkgDir / "library" / "libsds.nim")
  copyHeader(pkgDir, outDir)

task libsdsIOS, "Build a symbol-isolated static libsds for iOS":
  let pkgDir = sdsSourceDir()
  let outDir = outputDir()
  let arch = envOrDefault("ARCH", "arm64")
  let cpu = if arch == "x86_64": "amd64" else: arch
  if cpu notin ["arm64", "amd64"]:
    raise newException(CatchableError, "Unsupported iOS architecture: " & arch)
  let clangArch = if cpu == "amd64": "x86_64" else: cpu
  let sdk = envOrDefault("IPHONE_SDK", "iphoneos")
  let (sdkPath, sdkExit) = gorgeEx("xcrun --sdk " & quoteShell(sdk) & " --show-sdk-path")
  if sdkExit != 0 or sdkPath.strip().len == 0:
    raise newException(CatchableError, "Could not locate iOS SDK: " & sdk)
  let cacheDir = outDir / "nimcache"
  let archive = outDir / "libsds.a"
  if fileExists(archive):
    rmFile archive
  if dirExists(cacheDir):
    rmDir cacheDir
  exec compilerCommand(outDir) & " --app:staticlib --os:ios --cpu:" & cpu &
    " --cc:clang --compileOnly:on " & quoteShell(pkgDir / "library" / "libsds.nim")
  let (nimBin, nimExit) = gorgeEx("which nim")
  if nimExit != 0:
    raise newException(CatchableError, "Nim compiler not found")
  let nimLibDir = parentDir(parentDir(nimBin.strip())) / "lib"
  if not fileExists(nimLibDir / "nimbase.h"):
    raise newException(CatchableError, "nimbase.h not found in " & nimLibDir)
  let minimum = envOrDefault("IOS_TARGET", "13.0")
  let minFlag = if sdk == "iphonesimulator": "-mios-simulator-version-min=" else: "-miphoneos-version-min="
  let flags = " -arch " & clangArch & " -isysroot " & quoteShell(sdkPath.strip()) &
    " -I" & quoteShell(nimLibDir) & " -fembed-bitcode " & minFlag & minimum &
    " -O2 -fvisibility=hidden -fno-common"
  var objects: seq[string]
  for file in listFiles(cacheDir):
    if file.endsWith(".c"):
      let obj = file.changeFileExt("o")
      exec "clang" & flags & " -c " & quoteShell(file) & " -o " & quoteShell(obj)
      objects.add(obj)
  if objects.len == 0:
    raise newException(CatchableError, "No generated iOS C sources")
  let objectList = outDir / "ios-objects.txt"
  writeFile(objectList, objects.join("\n"))
  let merged = outDir / "libsds_merged.o"
  exec "xcrun ld -r -arch " & clangArch & " -exported_symbol '_sds_*' -o " &
    quoteShell(merged) & " -filelist " & quoteShell(objectList)
  let (commons, _) = gorgeEx("nm " & quoteShell(merged) & " | grep ' C ' || true")
  if commons.strip().len > 0:
    raise newException(CatchableError, "libsds leaks common symbols:\n" & commons)
  exec "ar rcs " & quoteShell(archive) & " " & quoteShell(merged)
  copyHeader(pkgDir, outDir)
