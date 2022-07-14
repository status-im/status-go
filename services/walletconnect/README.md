WalletConnect Session Service
================

WalletConnect Session service provides read/write API to fetch,store and delete wallet connect sessions.

To enable include walletconnect config part and add `walletconnect` to APIModules:


```json
{
  "WalletConnectConfig": {
    "Enabled": true,
  },
  APIModules: "walletconnect"
}
```

API
---

Enabling service will expose three additional methods:

#### walletconnect_storeWalletConnectSession

Stores newly created wallet connect session to the database
All fields are specified below:

```json
{
  "peer-id": "0643983b-0000-2222-1111-b05fdac338zd",
  "connector-info": "{:connected true,:accounts #js [0x3Ed3ab4A64C7D412bF628aDe9722c910ab20cE86], :chainId 1, :bridge https://c.bridge.walletconnect.org, :key c4ae6c97875ab90e64678f8fbeaeff5e38408f0d6ea3f58628556bc25bcc5092, :clientId 0643983b-0000-2222-1111-b05fdac338zd, :clientMeta #js {:name Status Wallet, :description Status is a secure messaging app, crypto wallet, and Web3 browser built with state of the art technology., :url #, :icons #js [https://statusnetwork.com/img/press-kit-status-logo.svg]}, :peerId 0643983b-0000-2222-1111-b05fdac338zd, :peerMeta #js {:name 1inch dApp, :description DeFi / DEX aggregator with the most liquidity and the best rates on Ethereum, Binance Smart Chain, Optimism, Polygon, 1inch dApp is an entry point to the 1inch Network's tech., :url https://app.1inch.io, :icons #js [https://app.1inch.io/assets/images/1inch_logo_without_text.svg https://app.1inch.io/assets/images/logo.png]}, :handshakeId 1657776235200377, :handshakeTopic 0643983b-0000-2222-1111-b05fdac338zd}"
}
```

#### walletconnect_fetchWalletConnectSessions

Finds all the stored wallet connect sessions and returns them
in an array format

TODO : Define the return json format

#### walletconnect_deleteWalletConnectSession

Finds the walletconnect session by peer-id and then deletes it
from the database