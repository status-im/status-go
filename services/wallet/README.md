Wallet API
==========

Wallet service starts a loop that watches for new transfers (eth and erc20).
To correctly start a service two values need to be changed in the config:

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

Wallet API
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
    "Header": {
      "parentHash": "0xd2130443688b760cb6710a8550ffe68106238a3103bf6e62f0784c9cb3e18591",
      "sha3Uncles": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
      "miner": "0x0000000000000000000000000000000000000000",
      "stateRoot": "0x420b2a421e99c28c3c07825c1ddac37e84cb55561b01c3702eb36a733499defe",
      "transactionsRoot": "0x00a1aaef3eac0928cd2ecd7c00a4562e5db92cf4763db9840ba449904616a2c8",
      "receiptsRoot": "0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2",
      "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
      "difficulty": "0x2",
      "number": "0x1",
      "gasLimit": "0x5ff7a7",
      "gasUsed": "0x5208",
      "timestamp": "0x5cebc0ed",
      "extraData": "0xd883010817846765746888676f312e31322e35856c696e75780000000000000064b57fc2cdf3d0318f02602a90160b749dd2813908a58d6428e695d438e749fc71f7528bee286fe8827db3bb3fccb1aa76593e81c4d9d5f1f7387f61f8b2854000",
      "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
      "nonce": "0x0000000000000000",
      "hash": "0x576123ac3d561b618d2c9730e5c4432af8d18ed151caeac8b166757ea3b060c7"
    },
    "Transaction": {
      "nonce": "0x0",
      "gasPrice": "0xa",
      "gas": "0xf4240",
      "to": "0x72058a9d5a8194078ed372b34fa1fb8b8e5b7720",
      "value": "0xde0b6b3a7640000",
      "input": "0x",
      "v": "0xa95",
      "r": "0xd171c582139765f44fa1401edcdd377ca1b350ee10c0685073a5d470fc3625c6",
      "s": "0x3392d24da2a13449345fbe8210d5af89f01ee21b456dc3c226e7cd7a0509fed",
      "hash": "0x56be9c3cf40243e62428a203e37ce2104cfa12bef7d9e47988fd769617d361c6"
    },
    "Receipt": {
      "root": "0x",
      "status": "0x1",
      "cumulativeGasUsed": "0x5208",
      "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
      "logs": [],
      "transactionHash": "0x56be9c3cf40243e62428a203e37ce2104cfa12bef7d9e47988fd769617d361c6",
      "contractAddress": "0x0000000000000000000000000000000000000000",
      "gasUsed": "0x5208"
    }
  }
]
```

Signals
-------

Two signals will are emmited:

1. Signal `newblock`

Emmited when transfers from new block were added to the database. In this case block number if the number of this new block.

```json
{
  "type": "wallet",
  "event": {
    "Type": "newblock",
    "BlockNumber": 10
  }
}
```

2. Signal `reorg`.

Emmited when part of blocks were removed. Starting from a given block number all transfers were removed.

```json
{
  "type": "wallet",
  "event": {
    "Type": "reorg",
    "BlockNumber": 10
  }
}
```