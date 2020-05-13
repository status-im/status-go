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

#### wallet_getTransfersByAddress

Returns avaiable transfers in a given range.

##### Parameters

- `address`: `HEX` - ethereum address encoded in hex
- `toBlock`: `BIGINT` - end of the range. if nil query will return last transfers.
- `limit`: `BIGINT` - limit of returned transfers.

##### Examples

```json
{"jsonrpc":"2.0","id":7,"method":"wallet_getTransfersByAddress","params":["0xb81a6845649fa8c042dfaceb3f7a684873406993","0x0","0x5"]}
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
