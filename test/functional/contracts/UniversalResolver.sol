// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

interface IENSRegistry {
    function resolver(bytes32 node) external view returns (address);
}

/// @title Minimal ENS Universal Resolver for Anvil functional tests
/// @notice The production ENS Universal Resolver lives at a fixed canonical
/// address on mainnet and its testnets, but is not deployed on Anvil. status-go
/// resolves ENS records (`addr`, `pubkey`, `contenthash`) through the Universal
/// Resolver, so the local test chain needs a contract answering those calls.
/// @dev This implements only the read path status-go uses: look up the name's
/// resolver in the well-known ENS registry and forward the encoded resolver call
/// to it, returning `(result, resolverAddress)` like the real Universal
/// Resolver's non-gateway `resolve(bytes,bytes)` overload. It does not implement
/// wildcard/ENSIP-10 or CCIP-read resolution, which the tests don't exercise.
contract UniversalResolver {
    // Canonical ENS registry address. The test infra syncs the deployed registry
    // here (see deploy_contracts.sh / sync_ens_registry.sh) so it can be read at a
    // stable address, matching status-go's expectation on every chain.
    IENSRegistry constant REGISTRY =
        IENSRegistry(0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e);

    function resolve(bytes calldata name, bytes calldata data)
        external
        view
        returns (bytes memory, address)
    {
        address resolver = REGISTRY.resolver(_namehash(name, 0));
        require(resolver != address(0), "UniversalResolver: resolver not found");
        (bool ok, bytes memory result) = resolver.staticcall(data);
        require(ok, "UniversalResolver: resolver call reverted");
        return (result, resolver);
    }

    // Reverse resolution is unused by the functional tests; kept to match the
    // interface status-go binds against.
    function reverse(bytes calldata, uint256)
        external
        pure
        returns (string memory, address, address)
    {
        return ("", address(0), address(0));
    }

    // Computes the ENS namehash of a DNS-encoded name (RFC1035 wire format:
    // length-prefixed labels terminated by a zero byte).
    function _namehash(bytes calldata name, uint256 offset)
        internal
        pure
        returns (bytes32)
    {
        uint8 length = uint8(name[offset]);
        if (length == 0) {
            return bytes32(0);
        }
        bytes32 parent = _namehash(name, offset + 1 + length);
        bytes32 labelHash = keccak256(name[offset + 1:offset + 1 + length]);
        return keccak256(abi.encodePacked(parent, labelHash));
    }
}
