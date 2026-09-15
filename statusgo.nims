import std/[os, strutils]

# libsds build orchestration. statusgo.nimble owns the nim-sds pin; every task
# here compiles the nim-sds copy this package's resolution (nimble.paths, from
# `nimble setup`) names, never any other checkout. To build against a local
# nim-sds, flip the requires line in statusgo.nimble to `file:///abs/path`
# and re-run `nimble setup` (never commit the flip; nimble 0.24.1's develop
# mode cannot override a URL#sha pin).
#
# After a one-time `nimble setup`:
#   nim libsds        statusgo.nims           # host shared libsds
#   nim libsdsIos     statusgo.nims           # iOS static lib
#   nim libsdsAndroid statusgo.nims           # ARCH + ANDROID_NDK_ROOT env
#
# The Go library itself is built by the Makefile (statusgo-shared-library and
# friends), which takes NIM_SDS_LIB_DIR / NIM_SDS_INC_DIR pointing at the
# artifacts these tasks produce, and STATUS_GO_BUILD_DIR for its own outputs.
#
# NOTHING IS WRITTEN INTO A SOURCE TREE. This package and nim-sds are normally
# READ-ONLY copies in nimble's package store, and both are built IN PLACE from
# there: every output goes under one caller-chosen directory. Three env vars are
# the whole contract:
#
#   STATUSGO_BUILD_DIR    root of all outputs. Default: this package's own
#                         directory (.sds-build).
#   STATUSGO_NIMBLE_PATHS the resolution to build against. Default: the
#                         nimble.paths beside this script. An embedder that has
#                         already resolved the whole graph points this at its
#                         own file instead of copying one into the store.
#   NIM_PARAMS            appended last to the inner nim-sds compiles (the
#                         channel nim-sds's tasks read), for anything else
#                         the caller needs.
#
# Artifact layout under STATUSGO_BUILD_DIR — embedders' -L/-I flags and rpaths
# depend on it:
#   .sds-build/build/libsds.*    the nim-sds artifacts
#   .sds-build/library/libsds.h  copy of nim-sds's API header (library/libsds.h)

proc absOut(p: string): string =
  if p.isAbsolute: p else: getCurrentDir() / p

proc isUnderStore(path: string): bool =
  ## Store copies live under nimble's package store (<nimbleDir>/pkgs2/…);
  ## any other resolved path is a develop-linked working copy.
  (DirSep & "pkgs2" & DirSep) in path

proc statusgoOutDir(): string =
  ## Root of every artifact these tasks produce; see the header. Relative
  ## values are resolved against the working directory, never against the
  ## package (a package-relative default would put outputs in the store).
  let env = getEnv("STATUSGO_BUILD_DIR")
  if env.len > 0:
    return absOut(env)
  if isUnderStore(thisDir()):
    quit "STATUSGO_BUILD_DIR is unset and this package is a read-only copy " &
      "in nimble's package store (" & thisDir() & "). Set it to a directory " &
      "of your own; the default only makes sense in a checkout."
  thisDir()

proc sdsArtifactDir(): string = statusgoOutDir() / ".sds-build" / "build"
proc sdsIncludeDir(): string = statusgoOutDir() / ".sds-build" / "library"

proc resolvedPathEntries(): seq[string] =
  ## This package's resolved dependency paths (--noNimblePath + --path flags,
  ## unquoted) from the nimble.paths that `nimble setup` generates, or from
  ## whatever file STATUSGO_NIMBLE_PATHS names. A bare --nimblePath is not an
  ## alternative: only exact --path entries pin the versions the resolution
  ## chose.
  var pathsFile = getEnv("STATUSGO_NIMBLE_PATHS")
  if pathsFile.len == 0:
    pathsFile = thisDir() / "nimble.paths"
  if not fileExists(pathsFile):
    quit "no dependency resolution to build against: " & pathsFile &
      " does not exist. In a checkout, run `nimble setup` first. An embedder " &
      "that has already resolved the graph points STATUSGO_NIMBLE_PATHS at " &
      "its own nimble.paths instead."
  for line in readFile(pathsFile).splitLines:
    let l = line.strip.replace("\"", "")
    if l == "--noNimblePath" or l.startsWith("--path:"):
      result.add l

proc sdsRootOf(path: string): string =
  ## The nim-sds package root a resolved path entry points into ("" if the
  ## entry is not an sds copy). Entries normally point at the package's
  ## srcDir `src`, so the root is usually the entry's parent.
  if fileExists(path / "sds.nimble"):
    return path
  if fileExists(parentDir(path) / "sds.nimble"):
    return parentDir(path)

proc sdsPackageRoot(): string =
  ## Root of the nim-sds copy in this package's resolution.
  for e in resolvedPathEntries():
    if not e.startsWith("--path:"):
      continue
    result = sdsRootOf(e["--path:".len .. ^1])
    if result.len > 0:
      return
  quit "nim-sds is not in this package's dependency resolution (no sds " &
    "entry in nimble.paths). Run `nimble setup` to materialize the " &
    "resolution. To build against a local nim-sds checkout, point the " &
    "requires line in statusgo.nimble at it (`file:///abs/path`, never " &
    "committed) and run `nimble setup` again."

proc depPathFlags(): string =
  ## The resolution as compiler flags for the inner sds compile. EVERY entry
  ## is kept, the sds one included: the copy being compiled IS the resolved
  ## copy, so there is no second sds on the path to split type identities, and
  ## the resolution's own entry is the authority on where sds's modules live
  ## (srcDir layout vs srcDir-hoisted).
  for e in resolvedPathEntries():
    result &= " " & e

proc copyIfChanged(src, dst: string) =
  ## Compare-before-copy: dependents relink only when bytes actually changed,
  ## so an unchanged rebuild must not touch the destination. Pure nimscript,
  ## no cmp/cp: on Windows `exec` goes through cmd.exe.
  mkDir parentDir(dst)
  if fileExists(dst) and readFile(dst) == readFile(src):
    return
  cpFile(src, dst)

proc runSdsTask(taskName: string, extraParams = "") =
  ## Runs a nim-sds build task against the resolved nim-sds copy, IN PLACE —
  ## including when that copy is read-only. nim-sds writes everything under
  ## SDS_OUT_DIR, so no scratch copy of the package is needed.
  let sdsRoot = sdsPackageRoot()
  # NIM_PARAMS is what nim-sds's tasks append, last, to every inner `nim c`.
  # --skipParentCfg:on: the --path entries below are the whole resolution; a
  # nim.cfg or config.nims above the nim-sds copy would add a second one.
  putEnv("NIM_PARAMS", "--skipParentCfg:on" & depPathFlags() &
    " " & extraParams & " " & getEnv("NIM_PARAMS"))
  putEnv("SDS_OUT_DIR", sdsArtifactDir())
  mkDir sdsArtifactDir()
  # nim-sds's task entry point: library/sds_tasks.nims includes the manifest
  # and dispatches its tasks without nimble. It ships with the package
  # (installDirs names library/), so a store copy has it as a checkout does.
  # sds.nimble is not an entry point: tasks only dispatch from a .nims.
  let entry = sdsRoot / "library" / "sds_tasks.nims"
  if not fileExists(entry):
    quit "the resolved nim-sds copy at " & sdsRoot & " has no" &
      " library/sds_tasks.nims. It predates the SDS_OUT_DIR contract these" &
      " tasks build against; bump the nim-sds pin in statusgo.nimble."
  exec "nim " & taskName & " " & quoteShell(entry)
  # The API header is the hand-written library/libsds.h in the nim-sds
  # package, what cgo compiles against; the libsds.h that --header leaves in
  # the nimcache is Nim's raw generated header, not the API. Mirror it next
  # to the artifacts so embedders read one fixed layout under
  # STATUSGO_BUILD_DIR and never need to know where the store put the sources.
  copyIfChanged(sdsRoot / "library" / "libsds.h", sdsIncludeDir() / "libsds.h")

# -d:noSignalHandler on every task: the Go runtime owns signal handling in
# the process libsds is loaded into. nim-sds's tasks pass it themselves too;
# the duplicate is harmless and keeps the requirement visible here.
task libsds, "Build libsds for the host desktop platform":
  const sdsTaskName =
    when defined(macosx): "libsdsDynamicMac"
    elif defined(linux): "libsdsDynamicLinux"
    else: "libsdsDynamicWindows"
  runSdsTask(sdsTaskName, "-d:noSignalHandler")

task libsdsIos, "Build libsds static library for iOS":
  runSdsTask("libsdsIOS", "-d:noSignalHandler")

task libsdsAndroid, "Build libsds for Android (ARCH + ANDROID_NDK_ROOT env)":
  # nim-sds has one task per target CPU, each setting ARCH itself in Nim's
  # naming; status-go's ARCH (Android ABI or Nim name) picks the task.
  let arch = getEnv("ARCH")
  let sdsTask =
    case arch
    of "arm64": "libsdsAndroidArm64"
    of "amd64", "x86_64": "libsdsAndroidAmd64"
    of "x86", "i386": "libsdsAndroidX86"
    of "arm": "libsdsAndroidArm"
    else:
      quit "ARCH must be one of arm64, amd64, x86_64, x86, i386, arm (got '" &
        arch & "')"
  runSdsTask(sdsTask, "-d:noSignalHandler")
