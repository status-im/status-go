import std/[os, strutils]

# libsds build orchestration. statusgo.nimble owns the nim-sds pin; every sds
# build task compiles the nimble-RESOLVED nim-sds copy — whatever this
# package's resolution (nimble.paths, from `nimble setup`) says — never any
# other checkout. To build against a patched/local nim-sds, substitute it the
# nimble-native way (`nimble develop --add:<path-to-nim-sds>` + `nimble
# setup`); the resolved copy then IS that checkout.
# After a one-time `nimble setup` (dependency materialization) build with:
#   nim libsds        statusgo.nims           # host desktop platform
#   nim libsdsIos     statusgo.nims           # iOS static lib
#   nim libsdsAndroid statusgo.nims           # ARCH + ANDROID_NDK_ROOT env
# Artifact locations follow the copy being built:
#   develop-linked checkout → built in place, artifacts in <checkout>/build,
#     header contract <checkout>/library/libsds.h (unchanged for embedders).
#   store copy (…/pkgs2/sds-…) → copied to .sds-build/ at the package root
#     (scratch dir, never installed) and built there, so the shared store
#     stays pristine. This is the `nimble install` case.

proc resolvedPathEntries(): seq[string] =
  ## This package's resolved dependency paths (--noNimblePath + --path flags,
  ## unquoted) from the nimble.paths that `nimble setup` generates.
  ## `nimble install` context: the package is built where no `nimble setup`
  ## ever ran. The outer nimble has already resolved and installed our
  ## dependencies into its store, so this nested setup only materializes the
  ## exact paths (no fresh solve, seconds not minutes). A bare --nimblePath
  ## fallback does NOT work here: lazy-path resolution proved unreliable for
  ## the inner compile, and only exact --path entries pin the versions the
  ## outer resolution chose.
  let pathsFile = thisDir() / "nimble.paths"
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
  ## srcDir `sds`, so the root is usually the entry's parent.
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

proc isUnderStore(path: string): bool =
  ## Store copies live under nimble's package store (<nimbleDir>/pkgs2/…);
  ## any other resolved path is a develop-linked working copy.
  (DirSep & "pkgs2" & DirSep) in path

proc depPathFlags(): string =
  ## The resolution as compiler flags for the inner sds compile, minus every
  ## sds entry: the prepared copy is authoritative, and a second sds copy on
  ## the path splits type identities.
  for e in resolvedPathEntries():
    if e.startsWith("--path:") and sdsRootOf(e["--path:".len .. ^1]).len > 0:
      continue
    result &= " " & e

proc prepareSdsSource(sdsRoot: string): string =
  ## The sds source tree the build tasks compile.
  if not isUnderStore(sdsRoot):
    return sdsRoot # develop-linked working copy → build in place
  # Store copy: never build inside the shared store — refresh a scratch copy.
  result = thisDir() / ".sds-build"
  if dirExists(result):
    exec "chmod -R u+w " & quoteShell(result)
    rmDir result
  cpDir sdsRoot, result
  # The store may be read-only; the build writes build/ + nimcache into the copy.
  exec "chmod -R u+w " & quoteShell(result)
  # Materialize the resolution inside the copy: sds's config.nims includes a
  # sibling nimble.paths when present, so the inner compile resolves its deps
  # even where NIMFLAGS forwarding is unavailable (unpatched upstream sds).
  var paths = ""
  for e in resolvedPathEntries():
    if e.startsWith("--path:"):
      if sdsRootOf(e["--path:".len .. ^1]).len > 0:
        continue
      paths &= "--path:\"" & e["--path:".len .. ^1] & "\"\n"
    else:
      paths &= e & "\n"
  writeFile(result / "nimble.paths", paths)

proc runSdsTask(taskName: string, extraEnvFlags = ""): string =
  ## Runs a nim-sds build task in the resolved copy; returns the directory
  ## that was built (artifacts in <dir>/build, header in <dir>/library).
  let sdsRoot = sdsPackageRoot()
  result = prepareSdsSource(sdsRoot)
  # `import sds` root: checkouts keep the srcDir layout (<root>/sds), while
  # copies installed by nimble's install action are srcDir-hoisted (modules
  # at the package root).
  let sdsImportPath =
    if dirExists(result / "sds"): result / "sds"
    else: result
  putEnv("NIMFLAGS", "--skipParentCfg:on" & depPathFlags() &
    " --path:" & quoteShell(sdsImportPath) & " " & extraEnvFlags &
    " " & getEnv("NIMFLAGS"))
  exec "cd " & quoteShell(result) & " && ln -sf sds.nimble sds.nims && nim " &
    taskName & " sds.nims"

task libstatus, "Build static libstatus + libsds for the host into build/bin (the auto-link layout)":
  # Produces the artifacts the status_go wrapper's auto-link flags reference:
  # build/bin/libstatus.a + build/bin/libsds.a. Static archives, so consumer
  # binaries survive package-store moves and need no rpath.
  # -d:noSignalHandler: the Go runtime owns signal handling in the host process.
  const sdsTaskName =
    when defined(macosx): "libsdsStaticMac"
    elif defined(windows): "libsdsStaticWindows"
    else: "libsdsStaticLinux"
  let srcDir = runSdsTask(sdsTaskName, "-d:noSignalHandler")
  let libDir = thisDir() / "build" / "bin"
  mkDir libDir
  let sdsLib = when defined(windows): "libsds.lib" else: "libsds.a"
  # libsds sits next to libstatus so the wrapper links both from one -L dir.
  cpFile srcDir / "build" / sdsLib, libDir / sdsLib
  exec "cd " & quoteShell(thisDir()) & " && make statusgo-library" &
    " NIM_SDS_LIB_DIR=" & quoteShell(libDir) &
    " NIM_SDS_INC_DIR=" & quoteShell(srcDir / "library") &
    " LIBSDS=" & quoteShell(libDir / sdsLib)

task libsds, "Build libsds for the host desktop platform":
  # -d:noSignalHandler: the Go runtime owns signal handling in the host process.
  const sdsTaskName =
    when defined(macosx): "libsdsDynamicMac"
    elif defined(linux): "libsdsDynamicLinux"
    else: "libsdsDynamicWindows"
  discard runSdsTask(sdsTaskName, "-d:noSignalHandler")

task libsdsIos, "Build libsds static library for iOS":
  # -d:noSignalHandler: libsds is embedded in the app process (nim-ffi 0.1.5+
  # enforces declaring signal ownership).
  discard runSdsTask("libsdsIOS", "-d:noSignalHandler")

task libsdsAndroid, "Build libsds for Android (ARCH + ANDROID_NDK_ROOT env)":
  # Normalize Android ABI names to the Nim CPU names nim-sds expects.
  let arch = getEnv("ARCH")
  if arch == "x86": putEnv("ARCH", "i386")
  elif arch == "x86_64": putEnv("ARCH", "amd64")
  # -d:noSignalHandler: libsds is loaded into the service process alongside
  # the Go runtime, which owns signal handling.
  discard runSdsTask("libsdsAndroid", "-d:noSignalHandler")
