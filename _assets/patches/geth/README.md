Status Patches for geth (go-ethereum)
=====================================

Status-go uses Status' fork of [go-ethereum](https://github.com/status-im/go-ethereum) as its dependency. As any other Go dependency `go-ethereum` code is vendored and stored in `vendor/` folder.

The reason why we use a fork is because we introduced a couple of differences that make it work better on mobile devices but not necessarily are suitable for all cases.

# Creating patches

Instructions for creating a patch from the command line:

1. Do changes in `vendor/github.com/ethereum/go-ethereum/`,
1. Go to the root `status-go` directory,
1. Create a patch `git diff --relative=vendor/github.com/ethereum/go-ethereum > _assets/patches/geth/0000-name-of-the-patch.patch`
1. Commit changes.

# Testing patches

To test a newly created patch, run:

```
$ git apply _assets/patches/geth/0000-name-of-the-patch.patch --directory vendor/github.com/ethereum/go-ethereum
```

And run `make statusgo` to compile it and `make test` to run unit tests.

# Updating fork with a patch

To make the patch available for everyone, it needs to be applied and pushed to remote git repository.

1. Clone [github.com/status-im/go-ethereum](https://github.com/status-im/go-ethereum) to `$GOPATH` and pull all changes,
1. From `github.com/status-im/status-go` run `GETH_VERSION=v1.8.14 ./_assets/patches/update-fork-with-patches.sh`,
1. Go to `github.com/status-im/go-ethereum` and verify the latest commit and tag `v1.8.14`,
1. If all is good push changes to the upstream:
```
$ git push origin patched/v1.8.14
$ git push -f v1.8.14
```
