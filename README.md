# Status bindings for go-ethereum
[![TravisCI Builds](https://img.shields.io/badge/TravisCI-URL-yellowgreen.svg?link=https://travis-ci.org/status-im/status-go)](https://travis-ci.org/status-im/status-go)
[![GoDoc](https://godoc.org/github.com/status-im/status-go?status.svg)](https://godoc.org/github.com/status-im/status-go) [![Master Build Status](https://img.shields.io/travis/status-im/status-go/master.svg?label=build/master)](https://github.com/status-im/status-go/tree/master) [![Develop Build Status](https://img.shields.io/travis/status-im/status-go/develop.svg?label=build/develop)](https://github.com/status-im/status-go/tree/develop)

# Docs
- [How To Build](https://www.notion.so/status/Building-status-go-f6b827dd1302436ba0575f4c543a352e)
- [Notes on Bindings](https://www.notion.so/status/Binding-notes-344f30ce0f2845a2b43e2de70931284a)
- [Status-go docs](https://www.notion.so/status/status-go-4fbe361e8e75484abeadadc80dd4dcdc)

# Intro
status-go is an underlying part of [Status](status.im) - a browser, messenger, and gateway to a decentralized world.

It's written in Go and requires Go 1.8 or above.

It uses Makefile to do most common actions. See `make help` output for available commands.

status-go uses [go-ethereum](https://github.com/ethereum/go-ethereum) with [some patches applied](./_assets/patches/geth) in it, located under [`vendor/`](./vendor/github.com/ethereum/go-ethereum) directory. See [geth patches README](./_assets/patches/geth/README.md) for more info.

# Build
There are two main modes status-go can be built:

 - standalone server
 - library to link for Android or iOS

Use following Makefile commands:

- `make statusgo` (builds binary into `build/bin/statusd`)
- `make statusgo-android`) (builds .aar file `build/android-16/aar`)
- `make statusgo-ios` and `make statusgo-ios-simulator` (builds iOS related artifacts in `build/os-9.3/framework`)

# Testing
To setup accounts passphrase you need to setup an environment variable: `export ACCOUNT_PASSWORD="secret_pass_phrase"`.

Make sure the dependencies are installed first by running:
```
make lint-install
make mock-install
```

To test fully statusgo, use:
```
make ci
```

To test statusgo using a given network by name, use:
```
make ci networkid=rinkeby
```

To test statusgo using a given network by number ID, use:
```
make ci networkid=3
```

If you have problems running tests on public network we suggest reading [e2e guide](t/e2e/README.md).

If you want to launch specific test, for instance `RPCSendTransactions`, use the following command:
```
go test -v ./geth/api/ -testify.m ^RPCSendTransaction$
```

Note `-testify.m` as [testify/suite](https://godoc.org/github.com/stretchr/testify/suite) is used to group individual tests.

# Licence
[Mozilla Public License 2.0](https://github.com/status-im/status-go/blob/develop/LICENSE.md)
