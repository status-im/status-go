Encrypting and decrypting Waku Messages
===

The Waku Message format provides an easy way to encrypt messages using symmetric or asymmetric encryption. The encryption comes with several handy design requirements: confidentiality, authenticity and integrity. You can find more details about Waku Message Payload Encryption in 26/WAKU-PAYLOAD.

## What data is encrypted
With Waku Message Version 1, the entire payload is encrypted.

Which means that the only discriminating data available in clear text is the content topic and timestamp (if present). Hence, if Alice expects to receive messages under a given content topic, she needs to try to decrypt all messages received on said content topic.

This needs to be kept in mind for scalability and forward secrecy concerns:

- If there is high traffic on a given content topic then all clients need to process and attempt decryption of all messages with said content topic;
- If a content topic is only used by a given (group of) user(s) then it is possible to deduce some information about said user(s) communications such as sent time and frequency of messages.

## Key management
By using Waku Message Version 1, you will need to provide a way to your users to generate and store keys in a secure manner. Storing, backing up and recovering key is out of the scope of this guide.

## Which encryption method should I use?
Whether you should use symmetric or asymmetric encryption depends on your use case.

Symmetric encryption is done using a single key to encrypt and decrypt.

Which means that if Alice knows the symmetric key `K` and uses it to encrypt a message, she can also use `K` to decrypt any message encrypted with `K`, even if she is not the sender.

Group chats is a possible use case for symmetric encryption: All participants can use an out-of-band method to agree on a `K`. Participants can then use `K` to encrypt and decrypt messages within the group chat. Participants MUST keep `K` secret to ensure that no external party can decrypt the group chat messages.

Asymmetric encryption is done using a key pair: the public key is used to encrypt messages, the matching private key is used to decrypt messages.

For Alice to encrypt a message for Bob, she needs to know Bob’s Public Key `K`. Bob can then use his private key `k` to decrypt the message. As long as Bob keep his private key `k` secret, then he, and only he, can decrypt messages encrypted with `K`.

Private 1:1 messaging is a possible use case for asymmetric encryption: When Alice sends an encrypted message for Bob, only Bob can decrypt it.

## Symmetric Encryption

### Encrypt a Message
To encrypt a message, assign the 32 byte symmetric key to a `KeyInfo`, and set its `Kind` to `node.Symmetric`. `node.EncodeWakuMessage` can then be used to encrypt the payload of a WakuMessage (that uses version `1`).

```go
package main

import (
    "fmt"

    "github.com/waku-org/go-waku/waku/v2/utils"
    "github.com/waku-org/go-waku/waku/v2/node"
    "github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

func main(){
    symKey := []byte{1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32}

    keyInfo := &node.KeyInfo{
        Kind: node.Symmetric,
        SymKey: symKey,
	    // PrivKey: Set a privkey if the message requires a signature
    }

    msg := &pb.WakuMessage{
        Payload:      []byte("Hello World"),
        Version:      1,
        ContentTopic: "/test/1/test/proto",
        Timestamp:    utils.GetUnixEpoch(),
    }

    if err := node.EncodeWakuMessage(msg, keyInfo); err != nil {
        fmt.Println("Error encrypting the message payload: ", err)
    }
}
```
The `WakuMessage` payload will be encrypted, and can be later broadcasted via Relay or Lightpush. It's also possible to not have to rely on a `WakuMessage` by creating instead a `node.Payload`, setting the payload in the `Data` attribute and the key in `Key`, and then use `payload.Encode(1)`  to encrypt your message, receiving a `byte[]` as a result.

The message can also be signed with a private key (`*ecdsa.PrivKey`) by setting it into the `PrivKey` attribute of the `KeyInfo` instance.

### Decrypting a Message
To decrypt a message, regardless of the protocol on which it was received, assign the 32 byte symmetric key to a `KeyInfo`, and set its `Kind` to `node.Symmetric`. 

```go
package main

import (
    "fmt"

    "github.com/waku-org/go-waku/waku/v2/utils"
    "github.com/waku-org/go-waku/waku/v2/node"
    "github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

func main(){
    symKey := []byte{1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32}

    keyInfo := &node.KeyInfo{
        Kind: node.Symmetric,
        SymKey: symKey,
    }

    // This message could have arrived via any of these protocols: relay, filter, store
    msg := &pb.WakuMessage{
        Payload:      ..., // Some encrypted payload
        Version:      1,
        ...
    }

    if err := node.DecodeWakuMessage(msg, keyInfo); err != nil {
        fmt.Println("Error decrypting the message payload: ", err)
    }

    fmt.Println(msg.Payload)
}
```

The `WakuMessage` payload will be decrypted and available in plain text. Messages that can't be decrypted will return an `error`.

> To have access to the message author's public key and signature (if available) of the message, `DecodePayload(msg, keyInfo)` can be used instead. It will return a `DecodedPayload` instance containing `PubKey` and `Signature` attributes, and it's not a destructive operation, so the `WakuMessage` protobuffer will not be modified.



## Asymmetric Encryption

### Encrypt a Message
To encrypt a message, assign an `*ecdsa.PublicKey` to a `KeyInfo`, and set its `Kind` to `node.Asymmetric`. `node.EncodeWakuMessage` can then be used to encrypt the payload of a WakuMessage (that uses version `1`).

```go
package main

import (
    "fmt"

    "github.com/waku-org/go-waku/waku/v2/utils"
    "github.com/waku-org/go-waku/waku/v2/node"
    "github.com/waku-org/go-waku/waku/v2/protocol/pb"
    "github.com/ethereum/go-ethereum/crypto"
)

func main(){
   

    keyInfo := &node.KeyInfo{
        Kind: node.Asymmetric,
        PubKey: ..., // The public key to encrypt the messages with
	    // PrivKey: Set a privkey if the message requires a signature
    }

    msg := &pb.WakuMessage{
        Payload:      []byte("Hello World"),
        Version:      1,
        ContentTopic: "/test/1/test/proto",
        Timestamp:    utils.GetUnixEpoch(),
    }

    if err := node.EncodeWakuMessage(msg, keyInfo); err != nil {
        fmt.Println("Error encrypting the message payload: ", err)
    }
}
```
The `WakuMessage` payload will be encrypted, and can be later broadcasted via Relay or Lightpush. It's also possible to not have to rely on a `WakuMessage` by creating instead a `node.Payload`, setting the payload in the `Data` attribute and the key in `Key`, and then use `payload.Encode(1)`  to encrypt your message, receiving a `byte[]` as a result.

The message can also be signed with a private key (`*ecdsa.PrivKey`) by setting it into the `PrivKey` attribute of the `KeyInfo` instance.

### Decrypting a Message
To decrypt a message that was encrypted with your public key, regardless of the protocol on which it was received, assign the `*ecdsa.PrivateKey` to `KeyInfo`, and set its `Kind` to `node.Asymmetric`. 

```go
package main

import (
    "fmt"

    "github.com/waku-org/go-waku/waku/v2/utils"
    "github.com/waku-org/go-waku/waku/v2/node"
    "github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

func main(){
    keyInfo := &node.KeyInfo{
        Kind: node.Symmetric,
        PrivKey: ..., // Your private key
    }

    // This message could have arrived via any of these protocols: relay, filter, store
    msg := &pb.WakuMessage{
        Payload:      ..., // Some encrypted payload
        Version:      1,
        ...
    }

    if err := node.DecodeWakuMessage(msg, keyInfo); err != nil {
        fmt.Println("Error decrypting the message payload: ", err)
    }

    fmt.Println(msg.Payload)
}
```

The `WakuMessage` payload will be decrypted and available in plain text. Messages that can't be decrypted will return an `error`.

> To have access to the message author's public key and signature (if available) of the message, `DecodePayload(msg, keyInfo)` can be used instead. It will return a `DecodedPayload` instance containing `PubKey` and `Signature` attributes, and it's not a destructive operation, so the `WakuMessage` protobuffer will not be modified.
