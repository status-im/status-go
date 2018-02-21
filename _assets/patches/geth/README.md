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

# Patches

- [`0000-accounts-hd-keys.patch`](./0000-accounts-hd-keys.patch) — adds support for HD extended keys (links/docs?)
- [`0004-whisper-notifications.patch`](./0004-whisper-notifications.patch) — adds Whisper notifications (need to be reviewed and documented)
- [`0006-latest-cht.patch`](./0006-latest-cht.patch) – updates CHT root hashes, should be updated regularly to keep sync fast, until proper Trusted Checkpoint sync is not implemented as part of LES/2 protocol.
- [`0009-whisper-envelopes-tracing.patch`](./0009-whisper-envelopes-tracing.patch) — adds Whisper envelope tracing (need to be reviewed and documented)
- [`0010-geth-17-fix-npe-in-filter-system.patch`](./0010-geth-17-fix-npe-in-filter-system.patch) - Temp patch for 1.7.x to fix a NPE in the filter system.
- [`0011-geth-17-whisperv6-70fbc87.patch`](./0011-geth-17-whisperv6-70fbc87.patch) - Temp patch for 1.7.x to update whisper v6 to the upstream version at the `70fbc87` SHA1.
- [`0014-whisperv6-notifications.patch`](./0014-whisperv6-notifications.patch) — adds Whisper v6 notifications (need to be reviewed and documented)
- [`0015-whisperv6-envelopes-tracing.patch`](./0015-whisperv6-envelopes-tracing.patch) — adds Whisper v6 envelope tracing (need to be reviewed and documented)

# Updating

When a new stable release of `go-ethereum` comes out, we need to upgrade our vendored copy. We use `dep` for vendoring, so for upgrading:

- Change target branch for `go-ethereum` in `Gopkg.toml`.
- `dep ensure -update github.com/ethereum/go-ethereum`
- `make dep-ensure`

This will ensure that dependency is upgraded and fully patched. Upon success, you can do `make vendor-check` after committing all the changes, in order to ensure that all changes are valid.
