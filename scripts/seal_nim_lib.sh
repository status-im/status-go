#!/usr/bin/env bash
# Usage: seal_nim_lib.sh <C API prefixes> <nimcache> <sealed archive>
# Seals a Nim static library so its runtime can't clash with another Nim
# library's in one link: every object and archive the compile left in
# <nimcache> (a fresh one) goes into one relocatable object in which only the
# symbols starting with one of the space-separated <prefixes> stay global.
# Mach-O is told by lipo; ELF and COFF use NIM_SEAL_LD, NIM_SEAL_OBJCOPY,
# NIM_SEAL_AR and NIM_SEAL_NM (default ld, objcopy, ar, nm), the target's tools
# for a cross build.
set -euo pipefail
read -r -a prefixes <<< "$1"
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
  # The API by name, not "_<prefix>*": that also matches the hidden symbols
  # with the prefix (Nim's own copies of the API procs), which older linkers
  # refuse to export.
  exports="$sealed.exports"
  tr '\n' '\0' < "$list" | xargs -0 xcrun nm -m 2>/dev/null |
    awk -v apis="${prefixes[*]}" 'BEGIN {n = split(apis, api, " ")}
      / external / && !/private external|\(undefined\)/ {for (i = 1; i <= n; i++) if (index($NF, "_" api[i]) == 1) {print $NF; next}}' |
    LC_ALL=C sort -u > "$exports"
  [ -s "$exports" ] || { echo "seal_nim_lib.sh: no global ${prefixes[*]} symbols in $nimcache" >&2; exit 1; }
  xcrun ld -r -arch "$arch" -exported_symbols_list "$exports" -filelist "$list" -o "$merged"
  rm -f "$list" "$exports"
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
  keep=()
  for prefix in "${prefixes[@]}"; do keep+=(--keep-global-symbol="${prefix}*"); done
  "${NIM_SEAL_OBJCOPY:-objcopy}" --wildcard "${keep[@]}" --localize-hidden "$merged"
  # GNU objcopy leaves g++'s unique symbols (C++ inline variables) global.
  uniques="$("${NIM_SEAL_NM:-nm}" -g --defined-only "$merged" | awk '$2 == "u"')"
  if [ -n "$uniques" ]; then
    printf 'seal_nim_lib.sh: unique symbols stay global (compile with -fno-gnu-unique):\n%s\n' "$uniques" >&2
    exit 1
  fi
fi
"${NIM_SEAL_AR:-ar}" rcs "$sealed" "$merged"
rm -f "$merged"
