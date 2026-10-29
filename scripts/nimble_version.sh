#!/usr/bin/env bash
# Usage: nimble_version.sh version|commit
# The version of a nimble store entry of status-go, which has no .git: the tag
# it was installed from (what `git describe --tags` gives on a tagged commit),
# else <status_go.nimble version>-<short commit>. Reads nimblemeta.json.
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
meta="$root/nimblemeta.json"
[ -f "$meta" ] || { echo "nimble_version: no $meta" >&2; exit 1; }
flat="$(tr -d ' \n\t' < "$meta")"
revision="$(sed -n 's/.*"vcsRevision":"\([0-9a-f]*\)".*/\1/p' <<< "$flat")"
[ -n "$revision" ] || { echo "nimble_version: no vcsRevision in $meta" >&2; exit 1; }
commit="${revision:0:10}"
case "${1:-version}" in
  commit) echo "$commit" ;;
  version)
    tag="$(sed -n 's/.*"specialVersions":\[\([^]]*\)\].*/\1/p' <<< "$flat" | tr ',' '\n' | tr -d '"' |
      sed -n 's/^#\(v[0-9][0-9A-Za-z.+-]*\)$/\1/p' | head -n 1)"
    if [ -n "$tag" ]; then
      echo "$tag"
    else
      version="$(sed -n 's/^version *= *"\(.*\)".*/\1/p' "$root/status_go.nimble" | head -n 1)"
      echo "${version:-0.0.0}-$commit"
    fi ;;
  *) echo "Usage: $0 version|commit" >&2; exit 2 ;;
esac
