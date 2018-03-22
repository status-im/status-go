# Status Patches for geth (go-ethereum) cross-compiled in Xgo
---

Status-go uses [go-ethereum](https://github.com/ethereum/go-ethereum) (**upstream**) as its dependency. When cross-compiling with Xgo, some headers or definitions are not available within the Xgo environment. In such a situation, we temporarily patch the sources before kicking the build in Xgo and revert them afterwards (this is taken care by the respective Makefile targets).

We try to minimize number and amount of changes in those patches as much as possible, and whereas possible, to contribute changes into the upstream.

# Creating patches

Instructions for creating a patch from the command line:

1. Enter the command line at the go-ethereum dependency root in vendor folder.
1. Create the patch:
    1. If you already have a commit that represents the change, find its SHA1 (e.g. `$COMMIT_SHA1`) and do `git diff $COMMIT_SHA1 > file.patch`
    1. If the files are staged, do `git diff --cached > file.patch`

# Patches

- [`0001-fix-duktapev3-missing-SIZE_MAX-def.patch`](./0001-fix-duktapev3-missing-SIZE_MAX-def.patch) — Adds patch to geth 1.8.0 dependency duktapev3, to address issue where SIZE_MAX is not defined in xgo for Android
- [`0002-remove-dashboard-collectData.patch`](./0002-remove-dashboard-collectData.patch) — Deletes the body of `collectData` in the `dashboard` package, since it will import the `gosigar` package which in turn includes a header (`libproc.h`) which is missing in the iOS environment in Xgo.
