# Release Process of status-go

Releases are only done manually, and are synced to Status App releases.

## Versioning

For details on status-go's versioning approach, including how we handle Go modules, generated files, and our release workflow, see [docs/versioning.md](./docs/versioning.md).

## Release branch

The release branch takes the form of `release/vA.B.x`, where `x` is hardcoded.
For example a valid release branch name is `release/v0.177.x` or `release/v10.7.x`.
Commits on this branch may be tagged as releases.

## Tagging versions

To tag a version, you should run the command:

`make tag-version` to create a tag for `HEAD`

or 

`make tag-version TARGET_COMMIT={hash}` to create a tag for a specific hash

You will have to then check the tag is correct, and push the tag:

`git push origin {tag_created}`


That can then be used as a stable tag.

## Releasing a version with generated files

Since https://github.com/status-im/status-go/pull/5878, we don't commit the generated files. This made `status-go`
not "go-gettable" anymore. This is almost never a problem, because the main client of `status-go` is the Status App,
which gets `status-go` as a Git submodule.

Nevertheless, there are cases when `status-go` is used as a dependency of another Go project, e.g. https://github.com/status-im/matterbridge. 
A workaround for such cases is described in https://github.com/status-im/status-go/pull/6594. 
