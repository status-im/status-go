mode = ScriptMode.Verbose

import std/[os, strutils]

# The package root, also when an embedder runs this file through an include
# wrapper (where thisDir() is the wrapper's directory).
const pkgRoot = currentSourcePath().parentDir()

### Package
version     = "0.0.1"
author      = "Status Research & Development GmbH"
description = "Status messaging and wallet backend, built as a C archive for status-app"
license     = "MPL-2.0"

# status-go has no Nim code: this package only resolves and builds its Nim libraries.
srcDir = "internal/nimble/src"


### Dependencies
# Any newer Nim is fine; the root's resolution (or the embedder's NIM) picks the compiler.
requires "nim >= 2.2.6"

# The nim-sds whose C ABI the Go bindings (pinned in go.mod) match.
requires "https://github.com/logos-messaging/nim-sds#v0.4.0-rc.6"

# The untagged chronos commit nim-ffi needs; the libsds compile uses this tree.
requires "chronos#0de7b335d0ad5557ad5ba71a4b7662f7b201750e"


### Tasks
# They follow the Nim library build contract (see nim/README.md). An embedder
# runs them without nimble: a wrapper `$LIB_OUT_DIR/status_go.nims` holding
# `include "<status_go>/status_go.nimble"`, then
# `nim --skipUserCfg --skipParentCfg <task>` on it with its compiler and NIM_PARAMS.

### Helpers

proc shellWords(params: string): seq[string] =
  ## NIM_PARAMS split with POSIX shell rules, quoting kept (it is passed on as is).
  var word = ""
  var quote = '\0'
  var escaped = false
  for c in params:
    if escaped:
      word.add c
      escaped = false
    elif quote != '\0':
      word.add c
      if c == quote: quote = '\0'
      elif c == '\\' and quote == '"': escaped = true
    elif c == '\\':
      word.add c
      escaped = true
    elif c in {'"', '\''}:
      quote = c
      word.add c
    elif c in Whitespace:
      if word.len > 0: result.add word
      word = ""
    else:
      word.add c
  if word.len > 0: result.add word

proc shellUnquote(word: string): string =
  ## The value a POSIX shell gives `word`.
  var quote = '\0'
  var i = 0
  while i < word.len:
    let c = word[i]
    if quote == '\'':
      if c == '\'': quote = '\0' else: result.add c
    elif c == '\\' and i + 1 < word.len and (quote == '\0' or word[i + 1] in {'"', '\\', '$', '`'}):
      inc i
      result.add word[i]
    elif quote == '"':
      if c == '"': quote = '\0' else: result.add c
    elif c in {'"', '\''}:
      quote = c
    else:
      result.add c
    inc i

proc resolutionPaths(): seq[string] =
  ## The `--path` directories of the resolution in use: the caller's, when its
  ## NIM_PARAMS has any (an embedder passes its resolution there), else ours,
  ## from nimble.paths in a checkout after `nimble setup`. nimble.paths holds
  ## Nim string literals (`--path:"..."`, strutils.escape), not shell words.
  for arg in shellWords(getEnv("NIM_PARAMS")):
    let value = shellUnquote(arg)
    if value.startsWith("--path:"):
      result.add value["--path:".len .. ^1]
  let own = pkgRoot / "nimble.paths"
  if result.len == 0 and fileExists(own):
    for line in readFile(own).splitLines():
      let line = line.strip()
      if line.startsWith("--path:"):
        result.add unescape(line["--path:".len .. ^1])

proc nimblePkgDir(name: string): string =
  ## Where the resolution in use put `name`: the first `--path` that is, or sits
  ## in, a directory carrying `name.nimble`. Never a global nimble store.
  for dir in resolutionPaths():
    for candidate in [dir, dir.parentDir()]:
      if fileExists(candidate / (name & ".nimble")):
        return candidate
  raise newException(CatchableError, name & " unresolved: pass its --path in NIM_PARAMS, " &
    "or run `nimble setup` in this checkout")

proc libOutDir(lib, platform: string): string =
  ## LIB_OUT_DIR, which the caller keeps separate per library and platform
  ## (libraries clean fixed names such as generated/ inside it); by default
  ## build/nim/<lib>/<platform>.
  result = getEnv("LIB_OUT_DIR")
  if result.len == 0:
    result = pkgRoot / "build" / "nim" / lib / platform
  elif not result.isAbsolute():
    result = getCurrentDir() / result
  if (DirSep & "pkgs2" & DirSep) in (result & DirSep):
    raise newException(CatchableError,
      "LIB_OUT_DIR=" & result & " is inside a nimble store; point it elsewhere")

proc libNimParams(outDir, pkgDir, src: string): string =
  ## Caller's NIM_PARAMS, then the flags status-go owns for every Nim library
  ## (nim/lib_params), then the resolution's `--path`s with the package's own
  ## moved to its copy (both would compile every module twice). The copy sits
  ## under a project tree, so parent configs are skipped.
  result = "--skipParentCfg"
  for arg in shellWords(getEnv("NIM_PARAMS")):
    if not shellUnquote(arg).startsWith("--path:"):
      result.add " " & arg
  for line in readFile(pkgRoot / "nim" / "lib_params").splitLines():
    if line.strip().len > 0:
      result.add " " & line.strip()
  result.add " " & quoteShellPosix("--nimcache:" & outDir / "nimcache")
  result.add " --noNimblePath"
  for dir in resolutionPaths():
    result.add " " & quoteShellPosix("--path:" & dir.replace(pkgDir, src))

proc copyPkg(outDir, name, pkgDir: string): string =
  ## Copies the resolved package to LIB_OUT_DIR/src/<name>, so its tasks can
  ## write next to their sources without touching the nimble store.
  ## TODO: drop once the libraries write to LIB_OUT_DIR themselves.
  result = outDir / "src" / name
  if dirExists(result):
    rmDir result
  mkDir result.parentDir()
  cpDir pkgDir, result
  # A checkout set up with `nimble setup` carries its own resolution, which the
  # library's config.nims would include over ours.
  for f in ["nimble.paths", "nimble.develop"]:
    rmFile result / f

proc pinCopyPaths(cfg, pkgDir, src: string) =
  ## Appends the resolution's `--path`s to the copy's nim.cfg, for library tasks
  ## that do not pass NIM_PARAMS to every compile (nim-sds iOS and Android).
  ## Without them the compile mixes the store's modules with the copy's.
  var lines = "\n--noNimblePath\n"
  for dir in resolutionPaths():
    lines.add "--path:" & escape(dir.replace(pkgDir, src)) & "\n"
  writeFile(cfg, (if fileExists(cfg): readFile(cfg) else: "") & lines)

proc useCompiler() =
  ## Library tasks run under the compiler nimble resolved (or the caller's NIM),
  ## also for their own `exec "nim c ..."`.
  let nim = getEnv("NIM", findExe("nim"))
  putEnv("NIM", nim)
  putEnv("PATH", nim.parentDir() & PathSep & getEnv("PATH"))

proc libScript(outDir, pkg, pkgDir: string, inPlace: bool): tuple[script, cwd: string] =
  ## A `.nims` wrapper that includes the library's `.nimble`, so plain `nim`
  ## runs its tasks (no second resolution by nimble). A library that writes only
  ## to LIB_OUT_DIR runs from its own package; otherwise from a copy of it.
  var nimble = pkgDir / pkg & ".nimble"
  result = (outDir / pkg & ".nims", outDir)
  if not inPlace:
    let src = copyPkg(outDir, pkg, pkgDir)
    nimble = src / pkg & ".nimble"
    result = (src / pkg & ".nims", src)
  mkDir result.cwd
  writeFile(result.script, "include \"" & nimble.replace("\\", "/") & "\"\n")

proc runNimTask(script, task, cwd: string) =
  useCompiler()
  withDir cwd:
    exec quoteShell(getEnv("NIM")) & " --skipUserCfg --skipParentCfg --hints:off " & task & " " & quoteShell(script)

### Tasks

# nim-sds names every platform's output libsds.*; DynlibFormat would drop the
# lib prefix on Windows.
const hostLib = "libsds." & (when defined(windows): "dll"
                             elif defined(macosx): "dylib"
                             else: "so")

# TODO: true once nim-sds writes only to LIB_OUT_DIR (nim-sds#113); its tasks
# then run from the store entry through the include wrapper instead of a copy.
const sdsWritesToLibOutDir = false

proc runSdsTask(taskName, lib, platform: string) =
  let outDir = libOutDir("sds", platform)
  let pkgDir = nimblePkgDir("sds")
  let (script, cwd) = libScript(outDir, "sds", pkgDir, sdsWritesToLibOutDir)
  let src = if sdsWritesToLibOutDir: pkgDir else: cwd
  putEnv("NIM_PARAMS", libNimParams(outDir, pkgDir, src))
  putEnv("LIB_OUT_DIR", outDir)
  if not sdsWritesToLibOutDir:
    pinCopyPaths(src / "library" / "nim.cfg", pkgDir, src)
  runNimTask(script, taskName, cwd)
  if not sdsWritesToLibOutDir:
    cpFile src / "build" / lib, outDir / lib
    cpFile src / "library" / "libsds.h", outDir / "libsds.h"
    if dirExists(src / "library" / "generated"):
      cpDir src / "library" / "generated", outDir / "generated"

proc archOrHost(): string = getEnv("ARCH", hostCPU)

task libsds, "Build the host libsds into LIB_OUT_DIR (default: build/nim/sds/<os>-<cpu>)":
  runSdsTask("libsds", hostLib, hostOS & "-" & archOrHost())

task libsdsAndroid, "Build libsds for Android into LIB_OUT_DIR; ARCH selects the architecture":
  runSdsTask("libsdsAndroid", "libsds.so", "android-" & getEnv("ARCH", "arm64"))

task libsdsIOS, "Build libsds for iOS into LIB_OUT_DIR":
  runSdsTask("libsdsIOS", "libsds.a", "ios-" & archOrHost())
