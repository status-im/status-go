# Dependency Management

[`dep`](https://github.com/golang/dep) is a tool of choice when it comes to dependency management.

## How we use `dep`

1. Transitive dependencies of `go-ethereum`. The most important thing for us is
   to be in-sync there. We want to reduce the regression scope.
   Hence, we pin down all the dependencies of `go-ethereum` with SHAs in `Gopkg.toml` when
   importing a new version of upstream. (This is considered a bad practice for
   `dep` but we are willing to take the risk to keep consistency with the upstream).
1. Exclusive `status-go` dependencies. The policy there is to keep them as
   fresh as possible. Hence, no constraints for them in the `toml` file.

## Installing `dep`

`go get -u github.com/golang/dep/cmd/dep`

## Docs (worth reading)

1. [README](https://github.com/golang/dep/blob/master/README.md)
1. [F.A.Q.](https://github.com/golang/dep/blob/master/docs/FAQ.md)

## Checking-out all dependencies

`dep ensure` - download all the dependencies based on `Gopkg.lock`.
`make dep-ensure` - ensure all patches are applied, too. **(Recommended)**

`Gopkg.lock` is kept inact if it is in-sync with `Gopkg.toml`. If the `toml`
file is changed, `dep ensure` will re-generate `Gopkg.lock` as well.

## Adding a new Dependency

(see [Adding a new dependency](https://github.com/golang/dep#adding-a-dependency))

1. `$ dep ensure -add github.com/foo/bar`
1. Commit changes.

## Updating a dependency

(see: [Changing a Dependency](https://github.com/golang/dep#changing-dependencies))

1. Update constraint in the `Gopkg.toml` file if needed.
2. Run `dep ensure -update github.com/foo/bar`
3. Commit changes.

## Updating all dependencies

`dep ensure -update`

## Updating `Geth`

Use the `update-geth` make target. For major releases, provide the GETH_BRANCH parameter. (e.g. `make update-geth GETH_BRANCH=release/1.9`). If there were any changes made, they will be committed while running this target.

## Committing changes

Make sure that you don't commit unnecessary changes to `Gopkg.toml` and
`Gopkg.lock`.

## Common issues

1. Relative imports and "Could not introduce package, as its subpackage does not contain usable Go code". See [this comment](https://github.com/golang/dep/issues/899#issuecomment-317904001) for more information.
