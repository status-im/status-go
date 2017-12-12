# Status Patches to for geth (go-ethereum)
---

Status-go uses [go-ethereum](https://github.com/ethereum/go-ethereum) as its dependency — like any normal Go package, stored in `vendor/` directory.


```
cd $GOPATH/src/github.com/status-im
git clone https://github.com/ethereum/go-ethereum

# switch to the latest 1.7 release branch (you may want to checkout the tag here as well)
git co release/1.7

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
``
