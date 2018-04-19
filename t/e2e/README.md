e2e
===

This package contains all e2e tests divided into subpackages which represents (or should represent) business domains like transactions, chat etc.

These tests are run against public testnets: Ropsten and Rinkeby.

e2e package contains a few utilities which are described in a [godoc](https://godoc.org/github.com/status-im/status-go/t/e2e).

### Flags

#### 1. `-network`
The `-network` flag is used to provide either a network id or network name which specifies the ethereum network to use
for running all test. It by default uses the `StatusChain` network.

#### Usage

First of all you need to export an ACCOUNT_PASSWORD environment variable. It should be a passphrase
that was used to generate accounts used in tests. If you don't know this variable for default accounts
you will have to create your own accounts and request some funds from rinkeby or ropsten faucet.
Please see Preparation section for details.

To use the `ropsten` network for testing using network name:

```bash
ACCOUNT_PASSWORD=test go test -v ./t/e2e/... -p=1 -network=ropsten
```

To use the `rinkeby` network with chain id `4` for testing:

```bash
ACCOUNT_PASSWORD=test go test -v ./t/e2e/... -p=1 -network=4
```

#### Preparation

You will need `geth` in your PATH. Please visit: https://www.ethereum.org/cli.
Once installed - generate 2 accounts and remember the passphrase for them, so run this command twice:

```bash
geth account new --keystore=static/keys/
Your new account is locked with a password. Please give a password. Do not forget this password.
Passphrase:
Repeat passphrase:
Address: {b6120ddd881593537c2bd4280bae509ec94b1a6b}
```

We expect that accounts will be named in a certain way:

```bash
pushd static/keys/
mv UTC--2018-01-26T13-46-53.657752811Z--b6120ddd881593537c2bd4280bae509ec94b1a6b test-account1.pk
mv UTC--2018-01-26T13-47-49.289567120Z--9f04dc05c4c3ec3b8b1f36f7d7d153f3934b1f07 test-account2.pk
popd
```

Update config for tests with new accounts `static/config/public-chain-accounts.json`:

```json
{
  "Account1": {
    "Address": "0xb6120ddd881593537c2bd4280bae509ec94b1a6b"
  },
  "Account2": {
    "Address": "0x9f04dc05c4c3ec3b8b1f36f7d7d153f3934b1f07"
  }
}
```

Embed keys as a binary data, you will need to install `npm` tool and web3.js lib:

```bash
make generate
```

As a final step request funds from faucet for a chosen network:

- [Rinkeby](https://faucet.rinkeby.io/)
- [Ropsten](http://faucet.ropsten.be:3001/)

Finally, you are ready to run tests!
