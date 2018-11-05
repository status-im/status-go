# Status bindings for go-ethereum

[![TravisCI Builds](https://img.shields.io/badge/TravisCI-URL-yellowgreen.svg?link=https://travis-ci.org/status-im/status-go)](https://travis-ci.org/status-im/status-go)
[![GoDoc](https://godoc.org/github.com/status-im/status-go?status.svg)](https://godoc.org/github.com/status-im/status-go) [![Master Build Status](https://img.shields.io/travis/status-im/status-go/master.svg?label=build/master)](https://github.com/status-im/status-go/tree/master) [![Develop Build Status](https://img.shields.io/travis/status-im/status-go/develop.svg?label=build/develop)](https://github.com/status-im/status-go/tree/develop)

# Docs

- [How To Build](https://docs.status.im/docs/build_status_go.html)
- [How To Contribute](CONTRIBUTING.md)

# License

[Mozilla Public License 2.0](https://github.com/status-im/status-go/blob/develop/LICENSE.md)

### Releasing

To create a release, first increase the `VERSION` file according to semantic versioning.

You can then build the artifacts for the specific platform.

Once done, you can run:

`make prepare-release`

and 

`make release release_branch={{release_branch}}`

Where `release_branch` is the branch you are targeting.
You will also need to specify some form of credentials, `GITHUB_TOKEN` environment variable for example.
