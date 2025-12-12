// SPDX-License-Identifier: Mozilla Public License 2.0

pragma solidity ^0.8.17;

import { IERC721 } from "@openzeppelin/contracts/token/ERC721/IERC721.sol";

contract CommunityOwnable {
    error CommunityOwnable_InvalidTokenAddress();
    error CommunityOwnable_NotAuthorized();

    address public immutable ownerToken;
    address public immutable masterToken;

    constructor(address _ownerToken, address _masterToken) {
        ownerToken = _ownerToken;
        masterToken = _masterToken;

        if (ownerToken == address(0) && masterToken == address(0)) {
            revert CommunityOwnable_InvalidTokenAddress();
        }
    }

    /// @dev Reverts if the msg.sender does not possess either an OwnerToken or a MasterToken.
    modifier onlyCommunityOwnerOrTokenMaster() {
        if (
            (ownerToken != address(0) && IERC721(ownerToken).balanceOf(msg.sender) == 0)
                && (masterToken != address(0) && IERC721(masterToken).balanceOf(msg.sender) == 0)
        ) {
            revert CommunityOwnable_NotAuthorized();
        }
        _;
    }
}
