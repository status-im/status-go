# Status Patches for geth (go-ethereum)
---

Status-go uses [go-ethereum](https://github.com/ethereum/go-ethereum) (**upstream**) as its dependency. As any other Go dependency `go-ethereum` code is vendored and stored in `vendor/` folder.

However, there are a few changes has been made to the upstream, that are specific to Status and should not be merged to the upstream. We keep those changes as a set of patches, that can be applied upon each next release of `go-ethereum`. Patched version of `go-ethereum` is available in vendor folder.

We try to minimize number and amount of changes in those patches as much as possible, and whereas possible, to contribute changes into the upstream.

# Creating patches

Instructions for creating a patch from the command line:

1. Enter the command line at the go-ethereum dependency root in vendor folder.
1. Create the patch:
    1. If you already have a commit that represents the change, find its SHA1 (e.g. `$COMMIT_SHA1`) and do `git diff $COMMIT_SHA1 > file.patch`
    1. If the files are staged, do `git diff --cached > file.patch`

# Updating patches

1. Tweak the patch file.
1. Run `make dep-ensure` to re-apply patches.

# Removing patches

1. Remove the patch file
1. Remove the link from [this README] (./README.md)
1. Run `make dep-ensure` to re-apply patches.

# Updating

When a new stable release of `go-ethereum` comes out, we need to upgrade our vendored copy. We use `dep` for vendoring, so for upgrading:

- Change target branch for `go-ethereum` in `Gopkg.toml`.
- `dep ensure -update github.com/ethereum/go-ethereum`
- `make dep-ensure`

This will ensure that dependency is upgraded and fully patched. Upon success, you can do `make vendor-check` after committing all the changes, in order to ensure that all changes are valid.
