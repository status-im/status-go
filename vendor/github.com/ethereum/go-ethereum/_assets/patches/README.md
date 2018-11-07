Status Patches for geth (go-ethereum)
=====================================

status-go uses [go-ethereum](https://github.com/status-im/go-ethereum) as its dependency. As any other Go dependency `go-ethereum` code is vendored and stored in `vendor/` folder.

However, there are a few changes has been made to the upstream, that are specific to Status and should not be merged to the upstream. We keep those changes as a set of patches, that can be applied upon each next release of `go-ethereum`. Patched version of `go-ethereum` is available in vendor folder.

We try to minimize number and amount of changes in those patches as much as possible, and whereas possible, to contribute changes into the upstream.

# Creating patches

Instructions for creating a patch from the command line:

1. Do changes in `vendor/github.com/ethereum/go-ethereum/`,
1. Go to the root `status-go` directory,
1. Create a patch `git diff --relative=vendor/github.com/ethereum/go-ethereum > _assets/patches/geth/0000-name-of-the-patch.patch`
1. Commit changes.

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
