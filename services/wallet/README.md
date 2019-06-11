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
    "type": "erc20",
    "address": "0x5dc6108dc6296b052bbd33000553afe0ea576b5e",
    "blockNumber": 5687981,
    "blockhash": "0xcc4553f125be0bc6cc974518368145fcf1344f41e5de238205db0a1c185ea2fc",
    "transaction": {
      "nonce": "0x57",
      "gasPrice": "0x3b9aca00",
      "gas": "0x44ba8",
      "to": "0xc55cf4b03948d7ebc8b9e8bad92643703811d162",
      "value": "0x0",
      "input": "0xcae9ca5100000000000000000000000039d16cdb56b5a6a89e1a397a13fe48034694316e0000000000000000000000000000000000000000000000015af1d78b58c40000000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000449134709e00000000000000000000000000000000000000000000000000000000000000010000000000000000000000005dc6108dc6296b052bbd33000553afe0ea576b5e00000000000000000000000000000000000000000000000000000000",
      "v": "0x29",
      "r": "0x124587e9c1d16d8bd02fda1221aefbfca8e2f4cd6300ed2077ebf736789179ab",
      "s": "0x4309fddc1226dacb877488221a439c4f97d77dc2c3f5c8ea51f34f42417d3bda",
      "hash": "0x259dd45c9c4d52137f32b7787e6e1fb6c9faf70ba40b8137bf66ba03abc0da00"
    },
    "receipt": {
      "root": "0x",
      "status": "0x1",
      "cumulativeGasUsed": "0x389e1e",
      "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000200000000020000000000000000000000000000000000004000000000000000200000000000000020000000000008000000000000000000000000000000000000000000000000020000000000002000000800000000100000000000000010000000000000000000400000000000000001000000000040000000400000000400000000020000000000000008000000000020000000010000000002000000000000020000000002000000000000000000000000000000000200000000000000000020000010000000000000000000000400000000000000000000000000000000000000",
      "logs": [
        {
          "address": "0xc55cf4b03948d7ebc8b9e8bad92643703811d162",
          "topics": [
            "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925",
            "0x0000000000000000000000005dc6108dc6296b052bbd33000553afe0ea576b5e",
            "0x00000000000000000000000039d16cdb56b5a6a89e1a397a13fe48034694316e"
          ],
          "data": "0x0000000000000000000000000000000000000000000000015af1d78b58c40000",
          "blockNumber": "0x56caad",
          "transactionHash": "0x259dd45c9c4d52137f32b7787e6e1fb6c9faf70ba40b8137bf66ba03abc0da00",
          "transactionIndex": "0x10",
          "blockHash": "0xcc4553f125be0bc6cc974518368145fcf1344f41e5de238205db0a1c185ea2fc",
          "logIndex": "0xd",
          "removed": false
        },
        {
          "address": "0xc55cf4b03948d7ebc8b9e8bad92643703811d162",
          "topics": [
            "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
            "0x0000000000000000000000005dc6108dc6296b052bbd33000553afe0ea576b5e",
            "0x000000000000000000000000ee55b1661fd24c4760d92026cedb252a5a0f2a4e"
          ],
          "data": "0x0000000000000000000000000000000000000000000000015af1d78b58c40000",
          "blockNumber": "0x56caad",
          "transactionHash": "0x259dd45c9c4d52137f32b7787e6e1fb6c9faf70ba40b8137bf66ba03abc0da00",
          "transactionIndex": "0x10",
          "blockHash": "0xcc4553f125be0bc6cc974518368145fcf1344f41e5de238205db0a1c185ea2fc",
          "logIndex": "0xe",
          "removed": false
        },
        {
          "address": "0x39d16cdb56b5a6a89e1a397a13fe48034694316e",
          "topics": [
            "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
            "0x0000000000000000000000000000000000000000000000000000000000000000",
            "0x0000000000000000000000005dc6108dc6296b052bbd33000553afe0ea576b5e",
            "0x0000000000000000000000000000000000000000000000000000000000000044"
          ],
          "data": "0x",
          "blockNumber": "0x56caad",
          "transactionHash": "0x259dd45c9c4d52137f32b7787e6e1fb6c9faf70ba40b8137bf66ba03abc0da00",
          "transactionIndex": "0x10",
          "blockHash": "0xcc4553f125be0bc6cc974518368145fcf1344f41e5de238205db0a1c185ea2fc",
          "logIndex": "0xf",
          "removed": false
        }
      ],
      "transactionHash": "0x259dd45c9c4d52137f32b7787e6e1fb6c9faf70ba40b8137bf66ba03abc0da00",
      "contractAddress": "0x0000000000000000000000000000000000000000",
      "gasUsed": "0x34f42"
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
