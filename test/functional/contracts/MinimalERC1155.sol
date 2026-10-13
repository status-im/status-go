// SPDX-License-Identifier: MIT
pragma solidity 0.8.12;

/// Minimal ERC-1155 used by the router functional test.
/// Mints token id 1 with balance 5 to `to`.
contract MinimalERC1155 {
    mapping(uint256 => mapping(address => uint256)) private _balances;

    constructor(address to) {
        _balances[1][to] = 5;
    }

    function balanceOf(address owner, uint256 id) external view returns (uint256) {
        return _balances[id][owner];
    }

    function balanceOfBatch(address[] calldata owners, uint256[] calldata ids)
        external
        view
        returns (uint256[] memory balances)
    {
        require(owners.length == ids.length, "length");
        balances = new uint256[](owners.length);
        for (uint256 i = 0; i < owners.length; i++) {
            balances[i] = _balances[ids[i]][owners[i]];
        }
    }

    function safeTransferFrom(
        address from,
        address to,
        uint256 id,
        uint256 amount,
        bytes calldata
    ) external {
        require(_balances[id][from] >= amount, "balance");
        _balances[id][from] -= amount;
        _balances[id][to] += amount;
    }
}
