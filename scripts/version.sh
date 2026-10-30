#!/usr/bin/env bash
# status-go's version: `git describe --tags` in a checkout, else (a nimble store
# entry, which has no .git) what nimble recorded at install time.
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [ -e "$root/.git" ]; then
  exec git -C "$root" describe --tags
fi
exec bash "$root/scripts/nimble_version.sh" version
