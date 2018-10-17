Status Patches for geth (go-ethereum)
=====================================

We keep changes in patches because it gives as a clear picture. In case of merge conflicts, thanks to patches we can easily figure out how the final code should look like.

## Syncing with upstream

When a new geth version is released, we need to merge it to an appropriate branch and apply patches.

The format of branches looks like this: `patched/1.8`, `patched/1.9`, and so on.

In order to sync the upstream, follow this instruction:
1. Revert existing patches: `$ _assets/patches/patcher -r`,
1. Merge a new release: `$ git merge upstream/v1.8.16` where `v1.8.16` is a tag with a new release,
1. Apply patches back: `$ _assets/patches/patcher`.

In the last step, some patches might be invalid. In such a case, they need to be fixed before proceeding.

## Creating patches

Instructions for creating a patch from the command line:

1. Do changes in `vendor/github.com/ethereum/go-ethereum/`,
1. Go to the root `status-go` directory,
1. Create a patch `git diff --relative=vendor/github.com/ethereum/go-ethereum > _assets/patches/geth/0000-name-of-the-patch.patch`
1. Commit changes.

## How to fix a patch?

TBD
