# Versioning in status-go

status-go follows a modified versioning strategy, influenced by the needs of both our project and the Go ecosystem.

## Why This Approach?

status-go serves a very specific purpose as a core library for Status apps (desktop and mobile), not as a general-purpose Go module. This influences our versioning decisions in several ways:

1. **Clarity over Convention**: While we respect semantic versioning principles, we've adapted them to better communicate what matters most to our apps - distinguishing between main releases and hotfixes.

2. **Go Module Compatibility Challenges**: Go's module system requires path changes for major version increments (v2+), which adds unnecessary complexity for our use case since we're primarily building a library for our own applications.

3. **Release Coordination**: Our versioning system is designed to facilitate coordination between status-go and the apps that depend on it, prioritizing clear communication about changes over strict semantic versioning rules.

4. **Simplicity**: By keeping the MAJOR version fixed and focusing on MINOR for main releases and PATCH for hotfixes, we maintain a simpler, more predictable versioning system for our teams.

## Current Versioning Approach

We use a simplified versioning approach based on [Semantic Versioning](https://semver.org/):

- We keep the MAJOR version fixed (we no longer bump it for breaking changes)
- We bump MINOR version for releases from the main branch
- We bump PATCH version for hotfixes on release branches

Version numbers are formatted as `vX.Y.Z` (e.g., `v10.26.0`).

> 💡 Yes, we may end up with versions like `v10.250.0` eventually. This is by design and works well for our use case.

## Tagging Versions

1. We use `git` tags to track the version of the library
2. Run `./_assets/scripts/version.sh` to get the current version
3. To create a new version tag:
   - Run `make tag-version` to create a tag for `HEAD`, or
   - Run `make tag-version TARGET_COMMIT={hash}` to create a tag for a specific hash
4. Don't forget to `git push origin {tag_created}` to publish the tag
5. The created tag can be used to link mobile client to the new version

## Release Branches

- Release branches follow the format `release/vX.Y.x` (e.g., `release/v10.26.x`)
- We create these branches as needed for desktop/mobile releases
- Tags on `develop` branch (with PATCH=0) are made as needed - this is used by mobile team who reference status-go by version tag
- Tags on `release/*` branches (with PATCH>0) are created manually when a patch release is needed for desktop/mobile

## Context & History

Originally, we used semantic versioning with MAJOR version 1, incrementing the MAJOR version for every breaking change ([PR #5829](https://github.com/status-im/status-go/pull/5829)). However, Go modules require that packages with a MAJOR version greater than 1 include a `/v{MAJOR}` suffix in their import paths. We decided against this approach because `status-go` is not intended to be used as an importable Go package, but rather as a shared library for [status-desktop](https://github.com/status-im/status-desktop) and [status-mobile](https://github.com/status-im/status-mobile). [(Issue #6049)](https://github.com/status-im/status-go/issues/6049)

Though we considered updating import paths ([PR #6557](https://github.com/status-im/status-go/pull/6557)), we ultimately chose not to, since this would add unnecessary complexity with few benefits.

Additionally, we have removed generated files from the main repository ([PR #5878](https://github.com/status-im/status-go/pull/5878)), making the default branch not directly go-gettable. This is acceptable for our workflow, as status-go is not designed for direct third-party `go get` usage.

## Final Decisions

- **No `/v{MAJOR}` Suffix:** We do not add the `/v{MAJOR}` module path suffix, even for versions > 1.
- **Fixed MAJOR Version:** We no longer bump the MAJOR version for breaking changes.
- **Generated Files Not Committed:** The main branches do not include generated files.
- **Intended Usage:** status-go is to be used as a shared library of Status apps, not as a general Go module.

## Using status-go as a Go Dependency (Workaround)

For rare cases where status-go needs to be used as an importable Go module (e.g., by projects such as [matterbridge](https://github.com/status-im/matterbridge)), we provide a workaround:

1. **Create a `generated/{version}` branch:**
   - Use the script `_assets/scripts/branch_version_generated.sh` to automate the process of creating a branch with generated files
   - This script:
     - Checks out a branch named `generated/{version}` based on the latest tag
     - Un-gitignores generated files
     - Runs code generation
     - Fixes up the version file
     - Commits and pushes the result

2. **Go Get from Commit Hash:**
   - To import status-go, use:  
     ```
     go get github.com/status-im/status-go@<commit-sha>
     ```
   - Use the commit hash from the generated branch, not the semantic release tag (e.g., not `v10.26.0`). This avoids Go's requirement for the `/v{MAJOR}` import path suffix.

## Future Changes

This solution is intended as the simplest for now. In the future, we may:
- Add `/v{MAJOR}` suffix support
- Commit generated files to the primary branches

But for the moment, given our goals and usage patterns, we see no compelling reason to pursue this.
