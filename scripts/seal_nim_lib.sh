#!/usr/bin/env bash
# Usage: seal_nim_lib.sh <C API prefix> <nimcache> <sealed archive>
# Seals a Nim static library so its runtime can't clash with another Nim
# library's in one link: every object and archive the compile left in
# <nimcache> (a fresh one) goes into one relocatable object in which only the
# symbols starting with <prefix> stay global.
# Mach-O is told by lipo; ELF and COFF use NIM_SEAL_LD, NIM_SEAL_OBJCOPY and
# NIM_SEAL_AR (default ld, objcopy, ar), the target's tools for a cross build.
set -euo pipefail
prefix="$1"
nimcache="$2"
sealed="$3"
objects=()
while IFS= read -r object; do objects+=("$object"); done < <(find "$nimcache" -maxdepth 1 -name '*.o' | LC_ALL=C sort)
archives=()
while IFS= read -r archive; do archives+=("$archive"); done < <(find "$nimcache" -name '*.a' | LC_ALL=C sort)
[ "${#objects[@]}" -gt 0 ] || { echo "seal_nim_lib.sh: no objects in $nimcache" >&2; exit 1; }
merged="$sealed.o"
rm -f "$sealed" "$merged"
mkdir -p "$(dirname "$sealed")"
if arch="$(lipo -archs "${objects[0]}" 2>/dev/null)"; then
  # ld -r keeps the platform and minimum OS the objects carry. Uninitialized
  # globals must not be common symbols (-fno-common), which ld -r keeps global.
  list="$sealed.inputs"
  printf '%s\n' "${objects[@]}" "${archives[@]+"${archives[@]}"}" > "$list"
  xcrun ld -r -arch "$arch" -exported_symbol "_${prefix}*" -filelist "$list" -o "$merged"
  rm -f "$list"
  commons="$(nm "$merged" | grep ' C ' || true)"
  if [ -n "$commons" ]; then
    printf 'seal_nim_lib.sh: common symbols would stay global:\n%s\n' "$commons" >&2
    exit 1
  fi
else
  inputs=("${objects[@]}")
  if [ "${#archives[@]}" -gt 0 ]; then
    inputs+=(--whole-archive "${archives[@]}" --no-whole-archive)
  fi
  "${NIM_SEAL_LD:-ld}" -r -d "${inputs[@]}" -o "$merged"
  # Hidden symbols that match the prefix (Nim's own copies of the API procs)
  # go local too, as ld -r -exported_symbol drops them on Mach-O.
  "${NIM_SEAL_OBJCOPY:-objcopy}" --wildcard --keep-global-symbol="${prefix}*" --localize-hidden "$merged"
fi
"${NIM_SEAL_AR:-ar}" rcs "$sealed" "$merged"
rm -f "$merged"
