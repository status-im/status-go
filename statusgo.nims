import std/[os, strutils]

# libsds + libstatus build orchestration. statusgo.nimble owns the nim-sds pin;
# every sds build task compiles the nim-sds copy this package's resolution
# (nimble.paths, from `nimble setup`) names, never any other checkout. To build
# against a local nim-sds, substitute it the nimble-native way: `nimble develop
# --add:<path-to-nim-sds>` + `nimble setup`.
#
# After a one-time `nimble setup`:
#   nim libstatus     statusgo.nims           # host libstatus.a + shared libsds
#   nim libsds        statusgo.nims           # host shared libsds
#   nim libsdsIos     statusgo.nims           # iOS static lib
#   nim libsdsAndroid statusgo.nims           # ARCH + ANDROID_NDK_ROOT env
#
# NOTHING IS WRITTEN INTO A SOURCE TREE. This package and nim-sds are normally
# READ-ONLY copies in nimble's package store, and both are built IN PLACE from
# there: every output goes under one caller-chosen directory. Three env vars are
# the whole contract:
#
#   STATUSGO_BUILD_DIR    root of all outputs. Default: this package's own
#                         directory (build/bin, .sds-build).
#   STATUSGO_NIMBLE_PATHS the resolution to build against. Default: the
#                         nimble.paths beside this script. An embedder that has
#                         already resolved the whole graph points this at its
#                         own file instead of copying one into the store.
#   NIMFLAGS              appended to the inner nim-sds compiles (nim-sds
#                         forwards it), for anything else the caller needs.
#
# Artifact layout under STATUSGO_BUILD_DIR — embedders' -L/-I flags and rpaths
# depend on it:
#   build/bin/libstatus.*      the Go library + its generated header
#   build/bin/statusgo-lib/    the generated cbindings entry point
#   .sds-build/build/libsds.*  the nim-sds artifacts
#   .sds-build/library/libsds.h  copy of nim-sds's committed header contract

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
proc statusgoBinDir(): string = statusgoOutDir() / "build" / "bin"

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
    "resolution, or develop-link a local checkout first: " &
    "`nimble develop --add:<path-to-nim-sds>` + `nimble setup`."

proc depPathFlags(): string =
  ## The resolution as compiler flags for the inner sds compile. EVERY entry
  ## is kept, the sds one included: the copy being compiled IS the resolved
  ## copy, so there is no second sds on the path to split type identities, and
  ## the resolution's own entry is the authority on where sds's modules live
  ## (srcDir layout vs srcDir-hoisted).
  for e in resolvedPathEntries():
    result &= " " & e

proc copyIfChanged(src, dst: string) =
  ## ADR-0003 compare-before-copy: dependents relink only when bytes actually
  ## changed, so an unchanged rebuild must not touch the destination.
  mkDir parentDir(dst)
  exec "cmp -s " & quoteShell(src) & " " & quoteShell(dst) &
    " || cp " & quoteShell(src) & " " & quoteShell(dst)

proc runSdsTask(taskName: string, extraEnvFlags = "") =
  ## Runs a nim-sds build task against the resolved nim-sds copy, IN PLACE —
  ## including when that copy is read-only. nim-sds writes everything under
  ## SDS_OUT_DIR, so no scratch copy of the package is needed.
  let sdsRoot = sdsPackageRoot()
  # --skipParentCfg:on: the --path entries below are the whole resolution; a
  # nim.cfg or config.nims above the nim-sds copy would add a second one.
  putEnv("NIMFLAGS", "--skipParentCfg:on" & depPathFlags() &
    " " & extraEnvFlags & " " & getEnv("NIMFLAGS"))
  putEnv("SDS_OUT_DIR", sdsArtifactDir())
  mkDir sdsArtifactDir()
  # nim-sds's task entry point. An installed copy only keeps
  # library/sds_tasks.nims: nimble 0.22.3 strips root files that installDirs
  # does not cover. A develop-linked checkout also has sds.nims, the
  # conventional name there. sds.nimble is not an entry point: tasks only
  # dispatch from a .nims.
  var entry = sdsRoot / "library" / "sds_tasks.nims"
  if not fileExists(entry):
    entry = sdsRoot / "sds.nims"
  if not fileExists(entry):
    quit "the resolved nim-sds copy at " & sdsRoot & " has no task entry" &
      " point (library/sds_tasks.nims or sds.nims). It predates the" &
      " SDS_OUT_DIR contract these tasks build against — bump the nim-sds pin" &
      " in statusgo.nimble."
  exec "nim " & taskName & " " & quoteShell(entry)
  # The header contract is a committed file in nim-sds's source tree. Mirror it
  # next to the artifacts so embedders read one fixed layout under
  # STATUSGO_BUILD_DIR and never need to know where the store put the sources.
  copyIfChanged(sdsRoot / "library" / "libsds.h", sdsIncludeDir() / "libsds.h")

proc makeCommon(): string =
  ## What every delegation to this package's Makefile must carry.
  ## GENERATE_PREREQ= : the generated Go sources the library build needs are
  ## committed (see docs/building.md), so no generator toolchain is required —
  ## and `make generate` would WRITE into this tree, which may be read-only.
  " STATUS_GO_BUILD_DIR=" & quoteShell(statusgoOutDir() / "build") &
    " GENERATE_PREREQ="

task libstatus, "Build libstatus.a + a shared libsds for the host into <out>/build/bin (the auto-link layout)":
  # Produces the artifacts the status_go wrapper's auto-link flags reference.
  # libstatus is a static archive, so it needs no rpath. libsds is the SHARED
  # flavour: libsdsStaticLinux and libsdsStaticWindows export the whole Nim
  # runtime, which collides with the runtime of any Nim program that links
  # them. (libsdsStaticMac localizes its symbols and does not; the static
  # archive stays the macOS-only path until nim-sds does the same elsewhere.)
  # -d:noSignalHandler: the Go runtime owns signal handling in the host process.
  const sdsTaskName =
    when defined(macosx): "libsdsDynamicMac"
    elif defined(windows): "libsdsDynamicWindows"
    else: "libsdsDynamicLinux"
  runSdsTask(sdsTaskName, "-d:noSignalHandler")
  let libDir = statusgoBinDir()
  mkDir libDir
  const sdsLib =
    when defined(macosx): "libsds.dylib"
    elif defined(windows): "libsds.dll"
    else: "libsds.so"
  # libsds sits next to libstatus so the wrapper links both from one -L dir.
  copyIfChanged(sdsArtifactDir() / sdsLib, libDir / sdsLib)
  exec "cd " & quoteShell(thisDir()) & " && make statusgo-library" &
    makeCommon() &
    " NIM_SDS_LIB_DIR=" & quoteShell(libDir) &
    " NIM_SDS_INC_DIR=" & quoteShell(sdsIncludeDir()) &
    " LIBSDS=" & quoteShell(libDir / sdsLib)

task libsds, "Build libsds for the host desktop platform":
  # -d:noSignalHandler: the Go runtime owns signal handling in the host process.
  const sdsTaskName =
    when defined(macosx): "libsdsDynamicMac"
    elif defined(linux): "libsdsDynamicLinux"
    else: "libsdsDynamicWindows"
  runSdsTask(sdsTaskName, "-d:noSignalHandler")

task libsdsIos, "Build libsds static library for iOS":
  # -d:noSignalHandler: libsds is embedded in the app process (nim-ffi 0.1.5+
  # enforces declaring signal ownership).
  runSdsTask("libsdsIOS", "-d:noSignalHandler")

task libsdsAndroid, "Build libsds for Android (ARCH + ANDROID_NDK_ROOT env)":
  # Normalize Android ABI names to the Nim CPU names nim-sds expects.
  let arch = getEnv("ARCH")
  if arch == "x86": putEnv("ARCH", "i386")
  elif arch == "x86_64": putEnv("ARCH", "amd64")
  # -d:noSignalHandler: libsds is loaded into the service process alongside
  # the Go runtime, which owns signal handling.
  runSdsTask("libsdsAndroid", "-d:noSignalHandler")
