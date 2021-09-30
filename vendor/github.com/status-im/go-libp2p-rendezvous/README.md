Rendezvous
=================

#### What is this?

Similar to status-im/rendezvous in using a smaller liveness TTL for records (20s), and not using UNREGISTER REQUEST, 
due to assuming that the TTL is very low (making it incompatible with libp2p original rendezvous spec). This module
is intended to be used in go-waku as a ambient peer discovery mechanism.

This module uses protobuffers instead of RLP, and does not use ENR ([Ethereum Node Records](https://eips.ethereum.org/EIPS/eip-778))
