// SPDX-License-Identifier: Mozilla Public License 2.0
pragma solidity ^0.8.17;

import { BaseToken } from "./BaseToken.sol";

contract MasterToken is BaseToken {
    constructor(
        string memory _name,
        string memory _symbol,
        string memory _baseTokenURI,
        address _ownerToken
    )
        BaseToken(_name, _symbol, type(uint256).max, true, false, _baseTokenURI, _ownerToken, address(0x0))
    { }
}
