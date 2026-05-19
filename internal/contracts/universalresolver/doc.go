// Package universalresolver provides a thin Go binding for the ENSv2
// Universal Resolver contract.
//
// Unlike the abigen-generated bindings elsewhere in internal/contracts, this
// binding is hand-written against the ABI string. Only resolve() and reverse()
// are exercised, so the full abigen surface is unnecessary.
//
// Per https://docs.ens.domains/resolvers/universal the Universal Resolver is
// an upgradable proxy owned by the ENS DAO, deployed at the same address on
// mainnet and supported testnets. It is the canonical entry point for ENS
// resolution under ENSv2 and transparently handles wildcard resolvers,
// NameWrapper, and CCIP-Read (ERC-3668) gateway round-trips when the supplied
// backend is CCIP-Read-aware.
package universalresolver
