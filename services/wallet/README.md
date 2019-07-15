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

```
[

  {
    "id": "0xac14e5fb9a81fd7d0517e51e23c4f3a8040459bfe0c4bee97b813db2d0438e2e",
    "type": "eth",
    "blockNumber": "0x1",
    "blockhash": "0x1471b02682f2308ce74314d89009251afb1f2d5dedc6835d069b1ad6edf98257",
    "timestamp": "0x5d25a873",
    "gasPrice": "0xa",
    "gasLimit": "0xf4240",
    "gasUsed": "0x5208",
    "nonce": "0x0",
    "input": "0x",
    "txStatus": "0x1",
    "txHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "value": "0xde0b6b3a7640000",
    "from": "0xd1c9bfa31ae8c085ba4672b165151245b9bfc25e",
    "to": "0x9dfc85106d84405a83271c2fe0cdfc1ca311a1f5",
    "contract": "0x0000000000000000000000000000000000000000"
  },
  {
    "id": "0x2629ee5f443d558ee4ae9e1cf202d76c04e262051b8d8acde7b766bb9d95068e",
    "type": "erc20",
    "blockNumber": "0x2",
    "blockhash": "0x046ad915b86a5eaa6026c8cdd09ea2f09fd3e603dd6e1ea86e8318f4a4b7d4e0",
    "timestamp": "0x5d25a88a",
    "gasPrice": "0x1",
    "gasLimit": "0xb0b8",
    "gasUsed": "0xb0b8",
    "nonce": "0x1",
    "txStatus": "0x1",
    "input": "0xa9059cbb000000000000000000000000f759c6683dfc5dad899eb86529dfaf4d0b25af1b0000000000000000000000000000000000000000000000000000000000000064",
    "txHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "value": "0x64",
    "from": "0xbd691e87d65b2857de55ac44598161ea135f73f6",
    "to": "0xf759c6683dfc5dad899eb86529dfaf4d0b25af1b",
    "contract": "0xd2439b0e20823e1e4c08df2d19c3b6a4c5f8f2d1"
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