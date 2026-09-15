import std/[os, strutils]

# libsds + libstatus build orchestration. statusgo.nimble owns the nim-sds pin;
# every sds build task compiles the nimble-RESOLVED nim-sds copy — whatever
# this package's resolution (nimble.paths, from `nimble setup`) says — never
# any other checkout. To build against a patched/local nim-sds, substitute it
# the nimble-native way (`nimble develop --add:<path-to-nim-sds>` + `nimble
# setup`); the resolved copy then IS that checkout.
# After a one-time `nimble setup` (dependency materialization) build with:
#   nim libstatus     statusgo.nims           # host libstatus.a + libsds.a
#   nim libsds        statusgo.nims           # host shared libsds
#   nim libsdsIos     statusgo.nims           # iOS static lib
#   nim libsdsAndroid statusgo.nims           # ARCH + ANDROID_NDK_ROOT env
#
# NOTHING IS WRITTEN INTO A SOURCE TREE. Both this package and nim-sds are
# normally resolved to READ-ONLY copies in nimble's package store, and both are
# built IN PLACE from there: every output goes under one caller-chosen
# directory. Three env vars are the whole contract:
#
#   STATUSGO_BUILD_DIR    root of all outputs. Default: this package's own
#                         directory, which reproduces the historical checkout
#                         layout (build/bin, .sds-build) exactly.
#   STATUSGO_NIMBLE_PATHS the resolution to build against. Default: the
#                         nimble.paths beside this script. An embedder that has
#                         already resolved the whole graph points this at its
#                         own file instead of copying one into the store.
#   NIMFLAGS              appended to the inner nim-sds compiles (nim-sds
#                         forwards it), for anything else the caller needs.
#
# Artifact layout under STATUSGO_BUILD_DIR (unchanged from what embedders read
# before, so their -L/-I flags and rpaths did not move):
#   build/bin/libstatus.*      the Go library + its generated header
#   build/bin/statusgo-lib/    the generated cbindings entry point
#   .sds-build/build/libsds.*  the nim-sds artifacts
#   .sds-build/library/libsds.h  copy of nim-sds's committed header contract

proc absOut(p: string): string =
  if p.isAbsolute: p else: getCurrentDir() / p

proc statusgoOutDir(): string =
  ## Root of every artifact these tasks produce; see the header. Relative
  ## values are resolved against the working directory, never against the
  ## package (a package-relative default would put outputs in the store).
  let env = getEnv("STATUSGO_BUILD_DIR")
  if env.len > 0: absOut(env) else: thisDir()

proc sdsArtifactDir(): string = statusgoOutDir() / ".sds-build" / "build"
proc sdsIncludeDir(): string = statusgoOutDir() / ".sds-build" / "library"
proc statusgoBinDir(): string = statusgoOutDir() / "build" / "bin"

proc isUnderStore(path: string): bool =
  ## Store copies live under nimble's package store (<nimbleDir>/pkgs2/…);
  ## any other resolved path is a develop-linked working copy.
  (DirSep & "pkgs2" & DirSep) in path

proc resolvedPathEntries(): seq[string] =
  ## This package's resolved dependency paths (--noNimblePath + --path flags,
  ## unquoted) from the nimble.paths that `nimble setup` generates, or from
  ## whatever file STATUSGO_NIMBLE_PATHS names.
  ## `nimble install` context: the package is built where no `nimble setup`
  ## ever ran. The outer nimble has already resolved and installed our
  ## dependencies into its store, so this nested setup only materializes the
  ## exact paths (no fresh solve, seconds not minutes). A bare --nimblePath
  ## fallback does NOT work here: lazy-path resolution proved unreliable for
  ## the inner compile, and only exact --path entries pin the versions the
  ## outer resolution chose.
  var pathsFile = getEnv("STATUSGO_NIMBLE_PATHS")
  if pathsFile.len == 0:
    pathsFile = thisDir() / "nimble.paths"
  if not fileExists(pathsFile):
    # The generated file must never land in the package store: this package's
    # own copy is read-only and shared between every consumer of the pin. Put
    # it in the output directory instead, which is the caller's to own.
    if isUnderStore(thisDir()):
      pathsFile = statusgoOutDir() / "nimble.paths"
  if not fileExists(pathsFile):
    # nimble refuses project actions when the current directory is under its
    # buildtemp (hook staging is deliberately not treated as a project), so
    # the nested setup cannot run in place during `nimble install`. Run it on
    # a manifest copy in a scratch dir outside buildtemp — same NIMBLE_DIR
    # (inherited env) → same store and exact paths — and bring the generated
    # nimble.paths back.
    let scratch = getTempDir() / "statusgo-nimble-setup"
    rmDir scratch
    mkDir scratch
    cpFile thisDir() / "statusgo.nimble", scratch / "statusgo.nimble"
    if fileExists(thisDir() / "nimble.lock"):
      cpFile thisDir() / "nimble.lock", scratch / "nimble.lock"
    exec "cd " & quoteShell(scratch) & " && nimble setup"
    # The generated file lists the root package (this one) at the scratch
    # path; point that entry back here before the scratch dir disappears.
    mkDir parentDir(pathsFile)
    writeFile(pathsFile,
      readFile(scratch / "nimble.paths").replace(scratch, thisDir()))
    rmDir scratch
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
  ## copy (nothing is staged anywhere any more), so there is no second sds on
  ## the path to split type identities — and the resolution's own entry is the
  ## authority on where sds's modules live (srcDir layout vs srcDir-hoisted).
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
  ## including when that copy is a read-only store copy. nim-sds writes
  ## everything under SDS_OUT_DIR (its own no-writes-in-the-tree contract), so
  ## no scratch copy of the package is made or needed.
  let sdsRoot = sdsPackageRoot()
  putEnv("NIMFLAGS", "--skipParentCfg:on" & depPathFlags() &
    " " & extraEnvFlags & " " & getEnv("NIMFLAGS"))
  putEnv("SDS_OUT_DIR", sdsArtifactDir())
  mkDir sdsArtifactDir()
  # nim-sds's task entry point. An INSTALLED copy (the normal case here) only
  # keeps library/sds_tasks.nims: nimble 0.22.3 strips root files that
  # installDirs does not cover, and installFiles does not rescue them. A
  # develop-linked checkout has both, and sds.nims is the conventional name
  # there. `nim <task> sds.nimble` is NOT a fallback — nim answers "invalid
  # command: libsdsDynamicLinux"; tasks only dispatch from a .nims.
  var entry = sdsRoot / "library" / "sds_tasks.nims"
  if not fileExists(entry):
    entry = sdsRoot / "sds.nims"
  if not fileExists(entry):
    quit "the resolved nim-sds copy at " & sdsRoot & " has no task entry" &
      " point (library/sds_tasks.nims or sds.nims). It predates the" &
      " SDS_OUT_DIR contract these tasks build against — bump the nim-sds pin" &
      " in statusgo.nimble."
  exec "nim " & taskName & " " & quoteShell(entry)
  # The header contract lives in nim-sds's SOURCE tree (library/libsds.h, a
  # committed file). Mirror it next to the artifacts so embedders read ONE
  # fixed layout under STATUSGO_BUILD_DIR and never have to know where the
  # package store put the sources.
  copyIfChanged(sdsRoot / "library" / "libsds.h", sdsIncludeDir() / "libsds.h")

proc makeCommon(): string =
  ## What every delegation to this package's Makefile must carry.
  ## GENERATE_PREREQ= : the generated Go sources the library build needs are
  ## committed (see AGENTS.md), so a consumer needs neither protoc nor mockgen
  ## — and `make generate` would try to WRITE into this tree, which for a store
  ## copy is exactly what this file exists to avoid.
  " STATUS_GO_BUILD_DIR=" & quoteShell(statusgoOutDir() / "build") &
    " GENERATE_PREREQ="

task libstatus, "Build static libstatus + libsds for the host into <out>/build/bin (the auto-link layout)":
  # Produces the artifacts the status_go wrapper's auto-link flags reference:
  # build/bin/libstatus.a + build/bin/libsds.a. Static archives, so consumer
  # binaries survive package-store moves and need no rpath.
  # -d:noSignalHandler: the Go runtime owns signal handling in the host process.
  const sdsTaskName =
    when defined(macosx): "libsdsStaticMac"
    elif defined(windows): "libsdsStaticWindows"
    else: "libsdsStaticLinux"
  runSdsTask(sdsTaskName, "-d:noSignalHandler")
  let libDir = statusgoBinDir()
  mkDir libDir
  let sdsLib = when defined(windows): "libsds.lib" else: "libsds.a"
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
