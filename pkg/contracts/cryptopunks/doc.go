// Package cryptopunks provides Go bindings for the CryptoPunks smart contract.
//
// CryptoPunks is a pre-ERC721 NFT implementation that uses a custom
// transferPunk(address,uint256) method instead of the standard transferFrom method
// for token transfers. CryptoPunks were launched in June 2017, before the ERC721
// standard was established, and use a modified ERC20-like approach.
//
// The bindings were generated using abigen from the official CryptoPunks contract ABI.
// Contract address: 0xb47e3cd837dDF8e4c57F05d70Ab865de6e193BBB (Ethereum Mainnet)
package cryptopunks
