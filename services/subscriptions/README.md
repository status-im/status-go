# Signal Subscriptions

This package implements subscriptions mechanics using [`singal`](../../signal) package.

It defines 3 new RPC methods in the `eth` namespace and 2 signals.

## Methods

###`eth_subscribeSignal`
Creates a new filter and subscribes to it's changes via signals.

Parameters: receives the method name and parameters for the filter that is created.

Example 1:
```json
{
  "jsonrpc": "2.0", 
  "id": 1,
  "method": "eth_subscribeSignal", 
  "params": ["eth_newPendingTransactionFilter", []]
}
```

Example 2:
```json
{
  "jsonrpc": "2.0", 
  "id": 2,
  "method": "eth_subscribeSignal", 
  "params": [
    "shh_newFilter",
    [{ "topics": ["0x12341234bf4b564f"] }]
  ]
}
```

Supported filters: `shh_newFilter`, `eth_newFilter`, `eth_newBlockFilter`, `eth_newPendingTransactionFilter`
(see [Ethereum documentation](https://github.com/ethereum/wiki/wiki/JSON-RPC) for repsective parameters).

Returns: error or `subscriptionID`.


###`eth_unsubscribeSignal`
Unsubscribes and removes one filter by it's ID.
Unsubscribing from a filter removes it.

Parameters: `subscriptionID` obtained from `eth_subscribeSignal`
Returns: error if something went wrong while unsubscribing.

### `eth_clearSignalSubscriptions`
Unsubscribes from all active subscriptions. This method is called automatically
when the node is stopped.

Returns null or error.


## Signals

1. Subscription data received

```json
{
  "type": "subscriptions.data",
  "event": {
    "subscription_id": "shh_0x01",
    "data": {
        <whisper envelope 01>,
        <whisper envelope 02>,
        ...
    }
}
```

2. Subscription error received

```json
{
  "type": "subscriptions.error",
  "event": {
    "subscription_id": "shh_0x01",
    "error_message": "can not find filter with id: 0x01"
  }
}
```

