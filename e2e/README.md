e2e
===

This package contains all e2e tests divided into subpackages which represents (or should represent) business domains like transactions, chat etc.

These tests are run against public testnets: Ropsten and Rinkeby.

e2e package contains a few utilities which are described in a [godoc](https://godoc.org/github.com/status-im/status-go/e2e).

### Flags

#### 1. `-network`
The `-network` flag is used to provide either a network id or network name which specifies the ethereum network to use
for running all test. It by default uses the `StatusChain` network.

#### Usage

To use the `ropsten` network for testing using network name:

```bash
go test -v ./e2e/... -network=ropsten
```

To use the `rinkeby` network with chain id `4` for testing:

```bash
go test -v ./e2e/... -network=4
```


## Run

`make test-e2e`
