# Contributing to status-go

First of all, thank you for taking time to add more value to Status, we really appreciate it!

If you just have a question, don't open an issue but rather ask us on our [Discord server](https://discord.gg/3Exux7Y)
## Starter Links

You just want to contribute something without reading tons of documentation, right? There're only a few useful links to start with.

How [status-mobile](github.com/status-im/status-mobile) uses us:
https://github.com/status-im/status-go/wiki/Notes-on-Bindings

Architecture: TBD in [#238](https://github.com/status-im/status-go/issues/238)

You can also discover more information in https://hackmd.io/s/SkZI0bXIb

## Workflow

1. Pick an [issue](https://github.com/status-im/status-go/issues) to work on and drop a line there that you're working on that.
2. Wait for an approve from one of core contributors.
3. Fork the project and work right in the `develop` branch.
4. Work on the issue and file a PR back into `develop`.
5. Wait until your PR is [reviewed](https://hackmd.io/s/B1AenvFU-) by 2 core developers and merged.

## Code Style

Please, note that we follow [Effective Go](https://golang.org/doc/effective_go.html) and
[CodeReviewComments](https://github.com/golang/go/wiki/CodeReviewComments) in our code.

## Keep history clean

1. Squash PR before merging.
   You can do it either with GitHub API by merging with "Squash and merge" or locally if you want to preserve your signature.
   It is ok to merge multiple commits with "Rebase and merge" if they are logically separate.

2. Preserve as much history as possible.
   If you need to re-name file use `git mv` - it will preserve git history.

## Commit format

We use a slight variation of [Conventional commits](https://www.conventionalcommits.org/en/v1.0.0/).

We enforce the usage of `!` for breaking changes, or `_` for non-breaking. The rationale is that
if we don't enforce one or the other, often devs will forget to add `!` to breaking changes.
Forcing to add one or the other, will also hopefully force devs to make a decision with each commit.
