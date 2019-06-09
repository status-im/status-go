# Dependency Management

Dependencies are managed by Go modules. They should be managed automatically and additionally we vendor all dependencies into `vendor/`. More on [Go Modules](https://github.com/golang/go/wiki/Modules).

## Adding and updating a dependency

1. `$ get get -u github.com/foo/bar`
2. `$ make vendor`
3. Commit changes.

## Updating go-ethereum

Remember that we use our fork of go-ethereum. It is included in the list of dependencies (`go.mod`) as a regular package: `github.com/ethereum/go-ethereum v1.8.27`, however, at the bottom of `go.mod` there is a replace directive. So, if you want to upgrade go-ethereum, these two lines need to be updated.

## Committing changes

Make sure that you don't commit unnecessary changes to `go.mod` and
`go.sum`.

## Common issues

1. Relative imports and "Could not introduce package, as its subpackage does not contain usable Go code". See [this comment](https://github.com/golang/dep/issues/899#issuecomment-317904001) for more information.
