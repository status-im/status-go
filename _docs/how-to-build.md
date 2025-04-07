# Build status-go

## Introduction

status-go is an underlying part of Status. It heavily depends on [go-ethereum](https://github.com/ethereum/go-ethereum/) which is [forked](https://github.com/status-im/go-ethereum) and slightly modified by us.

## Build status-go

### 1. Requirements

* Nix (Installed automatically)
* Docker (only if cross-compiling).

> go is provided by Nix

### 2. Clone the repository

```shell
git clone https://github.com/status-im/status-go
cd status-go
```

### 3. Set up build environment

status-go uses nix in the Makefile to provide every tools required.

### 4. Build the status-backend

To get started, let’s build the Ethereum node Command Line Interface tool, called `statusd`.

```shell
make status-backend
```

Once that is completed, you can start it straight away by running
```shell
./build/bin/status-backend --address=localhost:12345
```

This will provide full API at http://localhost:12345. \
Checkout [`status-backend docs`](../cmd/status-backend/README.md) for more details.

### 5. Build a library for Android and iOS

```shell
make install-gomobile
make statusgo-cross # statusgo-android or statusgo-ios to build for specific platform
```

## Debugging

### IDE Debugging

If you’re using Visual Studio Code, you can rename the [.vscode/launch.example.json](https://github.com/status-im/status-go/blob/develop/.vscode/launch.example.json) file to .vscode/launch.json so that you can run the statusd server with the debugger attached.

### Android debugging

In order to see the log files while debugging on an Android device, do the following:

* Ensure that the app can write to disk by granting it file permissions. For that, you can for instance set your avatar from a file on disk.
* Connect a USB cable to your phone and make sure you can use adb.
Run

```shell
adb shell tail -f sdcard/Android/data/im.status.ethereum.debug/files/Download/geth.log
```

## Linting

```shell
make lint
```

## Testing

Next, run unit tests:

```shell
make test
```

Unit tests can also be run using `go test` command. If you want to launch specific test, for instance `RPCSendTransactions`, use the following command:

```shell
go test -tags gowaku_skip_migrations -v ./api/ -testify.m ^RPCSendTransaction$
```

Note -testify.m as [testify/suite](https://godoc.org/github.com/stretchr/testify/suite) is used to group individual tests.

To run a single test in a test suite (e.g. `TestTransferringKeystoreFiles`, which is part of `SyncDeviceSuite`):
```shell
go test -tags gowaku_skip_migrations -v ./server/pairing -test.run TestSyncDeviceSuite -testify.m ^TestTransferringKeystoreFiles$
```

Note: `TestSyncDeviceSuite` is not the name of the test suite, but the name of the test function that runs the `SyncDeviceSuite` suite.