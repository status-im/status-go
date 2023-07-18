// SPDX-License-Identifier: Mozilla Public License 2.0
pragma solidity ^0.8.17;

import "./BaseToken.sol";
import "./MasterToken.sol";

contract OwnerToken is BaseToken {
    event MasterTokenCreated(address masterToken);

    bytes public signerPublicKey;

    constructor(
        string memory _name,
        string memory _symbol,
        string memory _baseTokenURI,
        string memory _masterName,
        string memory _masterSymbol,
        string memory _masterBaseTokenURI,
        bytes memory _signerPublicKey
    ) BaseToken(
        _name,
        _symbol,
        1,
        false,
        true,
        _baseTokenURI,
        address(this),
        address(this))
    {
        signerPublicKey = _signerPublicKey;
        MasterToken masterToken = new MasterToken(_masterName, _masterSymbol, _masterBaseTokenURI, address(this));
        emit MasterTokenCreated(address(masterToken));
        address[] memory addresses = new address[](1);
        addresses[0] = msg.sender;
        _mintTo(addresses);
    }

    function setMaxSupply(uint256 _newMaxSupply) override external onlyOwner {
        revert("max supply locked");
    }

    function setSignerPublicKey(bytes memory _newSignerPublicKey) external onlyOwner {
        signerPublicKey = _newSignerPublicKey;
    }
}
