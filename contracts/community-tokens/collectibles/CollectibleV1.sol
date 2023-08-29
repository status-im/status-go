// SPDX-License-Identifier: Mozilla Public License 2.0
pragma solidity ^0.8.17;

import "./BaseToken.sol";

contract CollectibleV1 is BaseToken {
    constructor(
        string memory _name,
        string memory _symbol,
        uint256 _maxSupply,
        bool _remoteBurnable,
        bool _transferable,
        string memory _baseTokenURI,
        address _ownerToken,
        address _masterToken
    )
        BaseToken(_name, _symbol, _maxSupply, _remoteBurnable, _transferable, _baseTokenURI, _ownerToken, _masterToken)
    { }
}
