# Release Process of status-go

The release process describes creating a release on Github. Each release consists of a new tag and assets which are builds for various environments but not only.

The process is automated and, even though it's possible to run manually, should always be triggered using [our Jenkins job](https://ci.status.im/job/status-go/job/manual/).

## Versioning

We use [semver](https://semver.org/) but as we do not have a stable version yet, it's a bit skewed.

We use `0` as the MAJOR version and bump only MINOR when there are breaking changes and PATCH when there are no breaking changes.

## Custom build

1. Go to [Jenkins job](https://ci.status.im/job/status-go/job/manual/), 
1. Leave "RELEASE" **unchecked** and use your branch name.

After successful build, open it (https://ci.status.im/job/status-go/job/manual/$BUILD_ID/) in a browser. Artifacts will have a random ID, for example `status-go-android-181221-143603-5708af.aar`, means that `181221-143603-5708af` is a version you can use in [status-mobile](https://github.com/status-im/status-mobile).


## Release branch

The release branch takes the form of `release/v0.y.x`, where `x` is hardcoded.
For example a valid release branch name is `release/v0.177.x` or `release/v0.188.x`.
Currently commits on this branch are not tagged and the branch name is used as a ref.

### Hotfixes

If an hotfix is necessary on the release branch (that happens after the app is released, and we need to push out a patched version), we historically tagged it using the format `release/v0.177.x+hotfix.1`.


The process over release branches is still in work since we still had few coordinated release between desktop and mobile, and we are still in the exploration phase.


## Tagging versions

To tag a version, you should run the command:

`make tag-version` to create a tag for `HEAD`

or 

`make tag-version TARGET_COMMIT={hash}` to create a tag for a specific hash

You will have to then check the tag is correct, and push the tag:

`git push origin {tag_created}`


That can then be used as a stable tag.
