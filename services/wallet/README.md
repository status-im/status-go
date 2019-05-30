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

##### Returns

List of objects like:


```json
[
  {
    "Type": "eth",
    "BlockNumber": 16,
    "BlockHash": "0xa38dd2618151ecb84242a9140d05419758341678058a698a0e39a935dbd916f8",
    "Transaction": {
      "nonce": "0xf",
      "gasPrice": "0xa",
      "gas": "0xf4240",
      "to": "0x6e9d4b3f183e0eb0512ece77bfa75bb399e9119d",
      "value": "0xde0b6b3a7640000",
      "input": "0x",
      "v": "0xa96",
      "r": "0x45effbfb66282671debf06c7ee44f7fb2ab2c4d774aaed6319ff0b3368c6861c",
      "s": "0x28acc80833af02ba038e27feb4c49f50bf77f19318efb700a0da63f9e8909d58",
      "hash": "0x2d600389c60293162bb466195a80386a2caf79b8ff0663ffe0e5725606c6d9ec"
    },
    "Receipt": {
      "root": "0x",
      "status": "0x1",
      "cumulativeGasUsed": "0x5208",
      "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
      "logs": [],
      "transactionHash": "0x2d600389c60293162bb466195a80386a2caf79b8ff0663ffe0e5725606c6d9ec",
      "contractAddress": "0x0000000000000000000000000000000000000000",
      "gasUsed": "0x5208"
    }
  }
]
```

Signals
-------

Three signals will are emitted:

1. `newblock` signal

Emitted when transfers from new block were added to the database. In this case block number if the number of this new block.
Client expected to request transfers starting from received block.

```json
{
  "type": "wallet",
  "event": {
    "Type": "newblock",
    "BlockNumber": 10
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
    "Type": "reorg",
    "BlockNumber": 10
  }
}
```

3. `history`  signal.

Emmited if new transfer in old block was found.
Client expected to request transfers starting from this new block till the earliest known block.

```json
{
  "type": "wallet",
  "event": {
    "Type": "history",
    "BlockNumber": 10
  }
}
```