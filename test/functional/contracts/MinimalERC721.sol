// SPDX-License-Identifier: MIT
pragma solidity 0.8.12;

/// Minimal ERC-721 used by the router functional test.
/// Mints token id 1 to `to` and supports the transfer the router packs.
contract MinimalERC721 {
    mapping(uint256 => address) private _owners;

    constructor(address to) {
        _owners[1] = to;
    }

    function ownerOf(uint256 tokenId) external view returns (address) {
        return _owners[tokenId];
    }

    function safeTransferFrom(address from, address to, uint256 tokenId) external {
        require(_owners[tokenId] == from, "not owner");
        _owners[tokenId] = to;
    }
}
