# Status Patches to for geth (go-ethereum)
---

Status-go uses [go-ethereum](https://github.com/ethereum/go-ethereum) (**upstream**) as its dependency. As any other Go dependency `go-ethereum` code is vendored and stored in `vendor/` folder.

However, there are a few changes has been made to the upstream, that are specific to Status and should not be merged to the upstream. We keep those changes as a set of patches, that can be applied upon each next release of `go-ethereum`. Patched version of `go-ethereum` is available in the [status-im/go-ethereum](https://github.com/status/go-ethereum) repo.

We try to minimize number and amount of changes in those patches as much as possible, and whereas possible, to contribute changes into the upstream.

# Patches



# Updating upstream version

When a new stable release of `go-ethereum` comes out, we need to upgrade our fork and vendored copy.

**Note: The process is completely repeatable, so it's safe to remove current `go-ethereum` directory, clone latest upstream version and apply patches from scratch.**

## How to update forked version
Make sure you have `status-go` in your `$GOPATH/src/github.com/status-im/` first.

```
# from scratch
rm -rf $GOPATH/src/github.com/status-im/go-ethereum
cd $GOPATH/src/github.com/status-im/
git clone https://github.com/ethereum/go-ethereum

# switch to the latest 1.7 release branch (you may want to checkout the tag here as well)
git co release/1.7

# update remote url to point to our fork repo
 git remote set-url origin git@github.com:status-im/go-ethereum.git

# apply patches
for patch in $GOPATH/src/github.com/status-im/status-go/geth-patches/*.patch;
do
    patch -p1 < $patch;
done
```

Once patches applied, you might want to inspect changes between current vendored version and newly patched version by this command:
```
diff -Nru -x "*_test.go" -x "vendor" -x ".git" -x "tests" -x "build" --brief $GOPATH/src/github.com/status-im/go-ethereum $GOPATH/src/github.com/status-im/status-go/vendor/github.com/ethereum/go-ethereum
```

# Vendor patched version
## Using `dep` tool

TBD

## Manually
This method should be used only while `dep` tool workflow is not set up.

```
# remove existing version from vendor
rm -rf $GOPATH/src/github.com/status-im/vendor/github.com/ethereum/go-ethereum/

# copy whole directory
cp -a $GOPATH/src/github.com/status-im/go-ethereum $GOPATH/src/github.com/status-im/status-go/vendor/github.com/ethereum/

# remove unneeded folders
cd $GOPATH/src/github.com/status-im/status-go/vendor/github.com/ethereum/go-ethereum
rm -rf .git tests build vendor

# remove _test.go files
find . -type f -name "*_test.go" -exec rm '{}' ';'
```
