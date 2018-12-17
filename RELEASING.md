# Release Process of status-go

The release process describes creating a release on Github. Each release consists of a new tag and assets which are builds for various environments but not only.

The process is automated and, even though it's possible to run manually, should always be triggered using [our Jenkins job](https://ci.status.im/job/status-go/job/parallel/).

## Versioning

We use [semver](https://semver.org/) but as we do not have a stable version yet, it's a bit skewed.

We use `0` as the MAJOR version and bump only MINOR when there are breaking changes and PATCH when there are no breaking changes.

Additionally, a pre-release can be created and the version can look like as complicated as this:
```
0.MINOR.PATCH-beta.INDEX.GIT_SHA
```

## Releasing from a branch

TODO: create a script that can do that instead of manual work.

1. Make sure that your branch is rebased on `develop`,
1. Change `VERSION` file content to `0.X.Y-beta.Z.$GIT_SHA` where `GIT_SHA` is the commit you want to release,
1. Go to [Jenkins job](https://ci.status.im/job/status-go/job/parallel/) and use your branch. NOTE: do **not** select "RELEASE".

**NOTE**: remember to change `VERSION` content back before merge.

## Releasing pre-release from develop

TODO: create a script that can do that instead of manual work.

1. Pull `develop` branch,
1. Bump `Z` (`0.X.Y-beta.Z`) in the current version (`VERSION` file),
1. Commit and push the change,
1. Go to [Jenkins job](https://ci.status.im/job/status-go/job/parallel/), select "RELEASE" and use `develop` branch.

## Releasing from develop (regular release)

TODO: create a script that can do that instead of manual work.

1. Pull `develop` branch,
1. Checkout a new branch `release/0.X`,
1. Remove `-beta.Z` suffix from the current version (`VERSION` file),
1. Commit and push the change,
1. Checkout develop branch, bump `X`, reset `Z` to `0` and commit to develop,
1. Go to [Jenkins job](https://ci.status.im/job/status-go/job/parallel/), select "RELEASE" and use `release/0.X` branch.

## Releasing from develop (a hot-fix)

TODO: create a script that can do that instead of manual work.

1. Checkout a release branch you want to hot fix (branch have a name `release/0.X`),
1. Bump `Y` (`0.X.Y`) in the current version (`VERSION` file),
1. Commit and push the change to `release/0.X` branch,
1. Go to [Jenkins job](https://ci.status.im/job/status-go/job/parallel/), select "RELEASE" and use `release/0.X` branch name.
