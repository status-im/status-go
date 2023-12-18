# Contributing Guidelines

Thank you for considering contributing to our project! We appreciate your time and effort to make this project better.

## Table of Contents
1. [Workflow](#workflow)
2. [Code Style](#code-style)
3. [Keep History Clean](#keep-history-clean)
4. [Testing](#testing)
   - [Test Validation](#test-validation)
   - [Area of Impact](#area-of-impact)
5. [Pull Request Description](#pull-request-description)
   - [Feature Flags](#feature-flags)
   - [Removing Feature Flags](#removing-feature-flags)
6. [Test Coverage](#test-coverage)
   - [Maintaining Test Coverage](#maintaining-test-coverage)
   - [Metrics for Test Coverage](#metrics-for-test-coverage)

## Workflow

1. Pick an [issue](https://github.com/status-im/status-go/issues) to work on and drop a line there that you're working on that.
2. Wait for approval from one of the core contributors.
3. Fork the project and work right in the `develop` branch.
4. Work on the issue and file a PR back into `develop`. Make sure the PR is assigned to yourself.
5. Wait until your PR is [reviewed](https://hackmd.io/s/B1AenvFU-) by 2 core developers and merged.

## Code Style

Please note that we follow [Effective Go](https://golang.org/doc/effective_go.html) and [CodeReviewComments](https://github.com/golang/go/wiki/CodeReviewComments) in our code.

## Keep History Clean

1. **Squash PR before merging**: You can do it either with GitHub API by merging with "Squash and merge" or locally if you want to preserve your signature. It is ok to merge multiple commits with "Rebase and merge" if they are logically separate.

2. **Preserve as much history as possible**: If you need to re-name a file, use `git mv` - it will preserve git history.

## Commit format

We use a slight variation of [Conventional commits](https://www.conventionalcommits.org/en/v1.0.0/).

We enforce the usage of `!` for breaking changes, or `_` for non-breaking. The rationale is that
if we don't enforce one or the other, often devs will forget to add `!` to breaking changes.
Forcing to add one or the other, will also hopefully force devs to make a decision with each commit.

## Testing

### Test Validation

Every Pull Request (PR) should include tests to validate its correctness and to test the features implemented. Preferably, use [Behavior-Driven Development (BDD)](https://en.wikipedia.org/wiki/Behavior-driven_development) principles for writing tests.

### Area of Impact

PRs should be well described, and the description should clearly specify the area of impact. This helps reviewers and maintainers understand the changes being made.
Please request manual QA if the PR is high-risk or it's large impact.

## Pull Request Description

### Feature Flags

For PRs introducing new features, especially those in high-risk areas, consider using feature flags. Feature flags allow features to be toggled on or off, providing a way to deploy code changes to production while controlling the visibility of new features.
For example, for messenger, you can use https://github.com/status-im/status-go/blob/develop/protocol/common/feature_flags.go.

### Removing Feature Flags

Once a feature has undergone testing and is ready for production use, the feature flag can be removed. Ensure that the removal is accompanied by a comprehensive update to documentation and release notes.

## Test Coverage

### Maintaining Test Coverage

Test coverage is vital for ensuring the stability and reliability of our project. Follow these guidelines:

1. Before submitting a PR, check the existing test coverage for the modified or new code.
2. Ensure that the new code is covered by relevant unit tests, integration tests, or other appropriate testing methods.
3. If modifying existing code, update or add tests to cover the changes.
4. If adding new features, include tests that demonstrate the correct functionality and handle edge cases.

### Metrics for Test Coverage

PRs should not decrease the overall coverage. If possible, aim to increase the overall coverage with each contribution.

Thank you for your contribution!

Happy coding!
>>>>>>> b0d37d2fa (Update CONTRIBUTING.md)
