# Release Process of status-go

The release process describes creating a release on Github. Each release consists of a new tag and assets which are builds for various environments but not only.

The process is automated and, even though it's possible to run manually, should always be triggered using [our Jenkins job](https://ci.status.im/job/status-go/job/parallel/).

## Versioning

We use [semver](https://semver.org/) but as we do not have a stable version yet, it's a bit skewed.

We use `0` as the MAJOR version and bump only MINOR when there are breaking changes and PATCH when there are no breaking changes.

## Custom build

1. Go to [Jenkins job](https://ci.status.im/job/status-go/job/parallel/), 
1. Leave "RELEASE" **unchecked** and use your branch name.

After successful build, open it (https://ci.status.im/job/status-go/job/parallel/$BUILD_ID/) in a browser. Artifacts will have a random ID, for example `status-go-android-181221-143603-5708af.aar`, means that `181221-143603-5708af` is a version you can use in [status-react](https://github.com/status-im/status-react).

## Releasing a new patch (no breaking changes or a hot-fix release)

TODO: create a script that can do that instead of manual work.

1. Checkout a release branch you want to release from (release branches have names like `release/0.X`),
1. Cherry-pick a commit you want to include OR merge `develop` branch,
1. Bump `Y` (`0.X.Y`) in the current version (`VERSION` file),
1. Commit and push the change to `release/0.X` branch,
1. Go to [Jenkins job](https://ci.status.im/job/status-go/job/parallel/), select "RELEASE" and use `release/0.X` branch name.

## Releasing a new version (breaking changes)

TODO: create a script that can do that instead of manual work.

1. Merge your PR to `develop` branch,
1. Pull `develop` branch locally,
1. Bump `X`, reset `Z` to `0` and commit to `develop` with a message "Bump version to 0.X.Y",
1. Checkout a new branch `release/0.X`,
1. Remove `-beta.Z` suffix from the current version (`VERSION` file),
1. Commit and push the change,
1. Go to [Jenkins job](https://ci.status.im/job/status-go/job/parallel/), select "RELEASE" and use `release/0.X` branch.
