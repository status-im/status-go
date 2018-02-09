# Status Patches to for geth (go-ethereum)
---

Status-go uses [go-ethereum](https://github.com/ethereum/go-ethereum) (**upstream**) as its dependency. As any other Go dependency `go-ethereum` code is vendored and stored in `vendor/` folder.

However, there are a few changes has been made to the upstream, that are specific to Status and should not be merged to the upstream. We keep those changes as a set of patches, that can be applied upon each next release of `go-ethereum`. Patched version of `go-ethereum` is available in the [status-im/go-ethereum](https://github.com/status/go-ethereum) repo.

We try to minimize number and amount of changes in those patches as much as possible, and whereas possible, to contribute changes into the upstream.

# Creating patches

Instructions for creating a patch from the command line:

1. Enter the command line at the status-im/go-ethereum repo root
1. Create the patch:
    1. If you already have a commit that represents the change, find its SHA1 (e.g. `$COMMIT_SHA1`) and do `git diff $COMMIT_SHA1 > file.patch`
    1. If the files are staged, do `git diff --cached > file.patch`

# Patches

- [`0000-accounts-hd-keys.patch`](./0000-accounts-hd-keys.patch) — adds support for HD extended keys (links/docs?)
- [`0002-les-api-status.patch`](./0002-les-api-status.patch) — adds StatusBackend into LES code (need to be inspected, some things can and should be done outside of les code
- [`0003-dockerfiles-wnode-swarm.patch`](./0003-dockerfiles-wnode-swarm.patch) — adds Dockerfiles (who uses this?)
- [`0004-whisper-notifications.patch`](./0004-whisper-notifications.patch) — adds Whisper notifications (need to be reviewed and documented)
- [`0006-latest-cht.patch`](./0006-latest-cht.patch) – updates CHT root hashes, should be updated regularly to keep sync fast, until proper Trusted Checkpoint sync is not implemented as part of LES/2 protocol.
- [`0007-README.patch`](./0007-README.patch) — update upstream README.md.
- [`0009-whisper-envelopes-tracing.patch`](./0009-whisper-envelopes-tracing.patch) — adds Whisper envelope tracing (need to be reviewed and documented)
- [`0010-geth-17-fix-npe-in-filter-system.patch`](./0010-geth-17-fix-npe-in-filter-system.patch) - Temp patch for 1.7.x to fix a NPE in the filter system.
- [`0011-geth-17-whisperv6-70fbc87.patch`](./0011-geth-17-whisperv6-70fbc87.patch) - Temp patch for 1.7.x to update whisper v6 to the upstream version at the `70fbc87` SHA1.

# Updating upstream version

When a new stable release of `go-ethereum` comes out, we need to upgrade our fork and vendored copy.

**Note: The process is completely repeatable, so it's safe to remove current `go-ethereum` directory, clone latest upstream version and apply patches from scratch.**

### Using existing fork repo (recommended)

#### I. In our fork at /status-im/go-ethereum.

1. Remove the local `develop` branch.

    ```bash
    git branch -D develop
    ```

1. Pull upstream release branch into `develop` branch.

    ```bash
    git pull git@github.com:ethereum/go-ethereum.git <release_branch>:develop
    ```
    In our case `<release_branch>` would be `release/1.7` because the current stable version is
    1.7.x.

1. Apply patches

    ```bash
    for patch in $GOPATH/src/github.com/status-im/status-go/_assets/patches/geth/*.patch;
    do
        patch -p1 < $patch;
    done
    ```

    Once patches applied, you might want to inspect changes between current vendored version and newly patched version by this command:
    ```bash
    diff -Nru -x "*_test.go" -x "vendor" -x ".git" -x "tests" -x "build" --brief $GOPATH/src/github.com/status-im/go-ethereum $GOPATH/src/github.com/status-im/status-go/vendor/github.com/ethereum/go-ethereum
    ```

1. Push `develop` branch to our remote, rewriting history

    ```bash
    git push -f origin develop
    ```

#### II. In status-go repository

1. Update vendored `go-ethereum` (note that we use upstream's address there, we override the download link to our fork address in `Gopkg.toml`)

    ```bash
    dep ensure --update github.com/ethereum/go-ethereum
    ```

    `Gopkg.lock` will change and files within `vendor/ethereum/go-ethereum`.

1. Run tests

    ```bash
    make ci
    ```

1. Commit & push changes, create a PR
