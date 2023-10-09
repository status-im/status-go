// SPDX-License-Identifier: Mozilla Public License 2.0
pragma solidity ^0.8.17;

import { BaseToken } from "./BaseToken.sol";
import { CommunityOwnable } from "../CommunityOwnable.sol";

contract OwnerToken is BaseToken {
    bytes public signerPublicKey;

    constructor(
        string memory _name,
        string memory _symbol,
        string memory _baseTokenURI,
        address _receiver,
        bytes memory _signerPublicKey
    )
        BaseToken(_name, _symbol, 1, false, true, _baseTokenURI, address(this), address(this))
    {
        signerPublicKey = _signerPublicKey;
        address[] memory addresses = new address[](1);
        addresses[0] = _receiver;
        _mintTo(addresses);
    }

    function setMaxSupply(uint256 _newMaxSupply) external override onlyCommunityOwnerOrTokenMaster {
        revert("max supply locked");
    }

    function setSignerPublicKey(bytes memory _newSignerPublicKey) external onlyCommunityOwnerOrTokenMaster {
        signerPublicKey = _newSignerPublicKey;
    }
}
