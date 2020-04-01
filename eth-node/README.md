# Abstraction for Ethereum node implementation

This package is a collection of interfaces, data types, and functions to make status-go independent from the go-ethereum implementation.

status-go is a wrapper around an Ethereum node. This package was created to have a possibility of selecting the underlying Ethereum node implementation, namely [go-ethereum](https://github.com/ethereum/go-ethereum) or [Nimbus](http://github.com/status-im/nimbus). The underlying implementation is selected using [Go build tags](https://golang.org/pkg/go/build/#hdr-Build_Constraints).

* `types` and `core/types` -- provide interfaces of node services, common data types, and functions,
* `bridge` -- provide implementation of interfaces declared in `types` using [go-ethereum](https://github.com/ethereum/go-ethereum) or [Nimbus](http://github.com/status-im/nimbus) in `geth` and `nimbus` directories respectively,
* `crypto` -- provide cryptographic utilities not depending on [go-ethereum](https://github.com/ethereum/go-ethereum),
* `keystore` -- provide a keystore implementation not depending on [go-ethereum](https://github.com/ethereum/go-ethereum).

Note: `crypto` and `keystore` are not finished by either depending on [go-ethereum](https://github.com/ethereum/go-ethereum) or not providing [Nimbus](http://github.com/status-im/nimbus) implementation.

## How to use it?

If you have a piece of code that depends on [go-ethereum](https://github.com/ethereum/go-ethereum), check out this package to see if there is a similar implementation that does not depend on [go-ethereum](https://github.com/ethereum/go-ethereum). For example, you want to decode a hex-string into a slice of bytes. You can do that using go-ethereum's `FromHex()` function or use equivalent from this package and avoid importing [go-ethereum](https://github.com/ethereum/go-ethereum). Thanks to this, your code fragment might be built with [Nimbus](http://github.com/status-im/nimbus).

## License

[Mozilla Public License 2.0](https://github.com/status-im/status-go/blob/develop/LICENSE.md)
