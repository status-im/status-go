Wallet
==========

Wallet service starts a loop that watches for new transfers (eth and erc20).
To correctly start the service two values need to be changed in the config:

1. Set Enable to true in WalletConfig

```json
{
  "WalletConfig": {
    "Enabled": true,
  }
}
```

2. And expose wallet API with APIModules

```
{
  APIModules: "eth,net,web3,peer,wallet",
}
```

API
----------

#### wallet_getTransfers

Returns avaiable transfers in a given range.

##### Parameters

- `start`: `BIGINT` - start of the range
- `end`: `BIGINT` - end of the range. if nil query will return all transfers from start.

##### Examples

```json
{"jsonrpc":"2.0","id":14,"method":"wallet_getTransfers","params":[0,20]}
{"jsonrpc":"2.0","id":14,"method":"wallet_getTransfers","params":[0,null]}
{"jsonrpc":"2.0","id":13,"method":"wallet_getTransfers","params":[0]}
```

##### Returns

List of objects like:

```json
[
  {
    "type": "eth",
    "address": "0xd448dbe70b62304fc157319e00a041eea238c5eb",
    "blockNumber": "0x1",
    "blockhash": "0x2c8f84bc61572e82b39c7dc6bf067d7e71e5d53e745b1174e1b7df4938df0053",
    "timestamp": "0x2332",
    "transaction": {
      "nonce": "0x0",
      "gasPrice": "0x1",
      "gas": "0x5208",
      "to": "0xd448dbe70b62304fc157319e00a041eea238c5eb",
      "value": "0x16345785d8a0000",
      "input": "0x",
      "v": "0xa95",
      "r": "0x73159b07b55d810b3898b60a0e3aed87e59e097be6bcae508a9b60e3e1f0ec3a",
      "s": "0x2b58524c9b96228e1e996a1e236a52e4a10beb54aad7c9ee1bf36b613f4d9cfb",
      "hash": "0x23da761563d8aa59398649df43a89a9ae3a7497861313674c401481b7400e8f9"
    },
    "from": "0x27bc544041e129501a6e6fb3c54cf6f12970b1e3",
    "receipt": {
      "root": "0x",
      "status": "0x1",
      "cumulativeGasUsed": "0x5208",
      "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
      "logs": [],
      "transactionHash": "0x23da761563d8aa59398649df43a89a9ae3a7497861313674c401481b7400e8f9",
      "contractAddress": "0x0000000000000000000000000000000000000000",
      "gasUsed": "0x5208"
    }
  }
]
```

##### Examples

```json
{"jsonrpc":"2.0","id":14,"method":"wallet_getTransfers","params":[0,20]}
{"jsonrpc":"2.0","id":14,"method":"wallet_getTransfers","params":[0,null]}
{"jsonrpc":"2.0","id":13,"method":"wallet_getTransfers","params":[0]}
```

#### wallet_getTransfersByAddress

Returns avaiable transfers in a given range.

##### Parameters

- `address`: `HEX` - ethereum address encoded in hex
- `start`: `BIGINT` - start of the range
- `end`: `BIGINT` - end of the range. if nil query will return all transfers from start.

##### Examples

```json
{"jsonrpc":"2.0","id":7,"method":"wallet_getTransfersByAddress","params":["0xb81a6845649fa8c042dfaceb3f7a684873406993","0x0"]}
```

##### Returns

Objects in the same format.

#### wallet_getTokensBalances

Returns tokens balances mapping for every account. See section below for the response example.

##### Parameters

- `accounts` `HEX` - list of ethereum addresses encoded in hex
- `tokens` `HEX` - list of ethereum addresses encoded in hex

```json
{"jsonrpc":"2.0","id":11,"method":"wallet_getTokensBalances","params":[["0x066ed5c2ed45d70ad72f40de0b4dd97bd67d84de", "0x0ed535be4c0aa276942a1a782669790547ad8768"], ["0x5e4bbdc178684478a615354d83c748a4393b20f0", "0x5e4bbdc178684478a615354d83c748a4393b20f0"]]}
```

#### Returns

First level keys accounts, second level keys are tokens.

```json
{
  "0x066ed5c2ed45d70ad72f40de0b4dd97bd67d84de": {
    "0x1dfb2099f936b3e98bfc9b7059a8fb04edcce5b3": 12,
    "0x5e4bbdc178684478a615354d83c748a4393b20f0": 12
  },
  "0x0ed535be4c0aa276942a1a782669790547ad8768": {
    "0x1dfb2099f936b3e98bfc9b7059a8fb04edcce5b3": 14,
    "0x5e4bbdc178684478a615354d83c748a4393b20f0": 14
  }
}
```

Signals
-------

Two signals can be emitted:

1. `newblock` signal

Emitted when transfers from new block were added to the database. In this case block number if the number of this new block.
Client expected to request transfers starting from received block.

```json
{
  "type": "wallet",
  "event": {
    "type": "newblock",
    "blockNumber": 0,
    "accounts": [
      "0x42c8f505b4006d417dd4e0ba0e880692986adbd8",
      "0x3129mdasmeo132128391fml1130410k312312mll"
    ]
  }
}
```

2. `reorg` signal.

Emitted when part of blocks were removed. Starting from a given block number all transfers were removed.
Client expected to request new transfers from received block and replace transfers that were received previously.

```json
{
  "type": "wallet",
  "event": {
    "type": "reorg",
    "blockNumber": 0,
    "accounts": [
      "0x42c8f505b4006d417dd4e0ba0e880692986adbd8"
    ]
  }
}
```

3. `history` signal

Emmited when historical transfers were downloaded. Block number will refer the first block where historical transfers
were found.

```json
{
  "type": "wallet",
  "event": {
    "type": "history",
    "blockNumber": 0,
    "accounts": [
      "0x42c8f505b4006d417dd4e0ba0e880692986adbd8"
    ]
  }
}
```