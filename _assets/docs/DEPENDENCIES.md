# Dependency Management
[`dep`](https://github.com/golang/dep) is a tool of choice when it comes to dependency management.

## How we use `dep`.

1. Transitive dependencies of `go-ethereum`. The most important thing for us is
   to be in-sync there. We want to reduce the regression scope.
   Hence, we pin down all the dependencies of `go-ethereum` with SHAs in `Gopkg.toml` when
   importing a new version of upstream. (This is considered a bad practice for
   `dep` but we are willing to take the risk to keep consitency with the upstream).

2. Exclusive `status-go` dependencies. The policy there is to keep them as
   fresh as possible. Hence, no constraints for them in the `toml` file.

## Installing `dep`

`go get -u github.com/golang/dep/cmd/dep`


## Docs (worth reading)
1. [README](https://github.com/golang/dep/blob/master/README.md)
2. [F.A.Q.](https://github.com/golang/dep/blob/master/docs/FAQ.md)


## Checking-out all dependencies

`dep ensure` - download all the dependencies based on `Gopkg.lock`.
`make dep-ensure` - ensure all patches are applied, too. **(Recommended)**

`Gopkg.lock` is kept inact if it is in-sync with `Gopkg.toml`. If the `toml`
file is changed, `dep ensure` will re-generate `Gopkg.lock` as well.


## Adding a new Dependency
(see [Adding a new dependency](https://github.com/golang/dep#adding-a-dependency))
1. `$ dep ensure -add github.com/foo/bar`
2. Commit changes.


## Updating a dependency
(see: [Changing a Dependency](https://github.com/golang/dep#changing-dependencies))
1. Update constraint in the `Gopkg.toml` file if needed.
2. Run `dep ensure -update github.com/foo/bar`
3. Commit changes.

## Updating all dependencies

`dep ensure -update`

## Updating `Geth`

1. (for major releases) Update the branch name in `Gopkg.toml`.
2. Update the `go-ethereum` dependency: `dep ensure -v -update github.com/ethereum/go-ethereum`.
3. Update vendor files in `status-go`, and apply patches by running `make dep-ensure`.
4. Commit patches.
5. Commit `Gopkg.lock`, `Gopkg.toml` and `vendor` directories.


## Commiting changes

Make sure that you don't commit unnecessary changes to `Gopkg.toml` and
`Gopkg.lock`.


## Common issues

1. Relative imports and "Could not introduce package, as its subpackage does not contain usable Go code". See [this comment](https://github.com/golang/dep/issues/899#issuecomment-317904001) for more information.
