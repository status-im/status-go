// SPDX-License-Identifier: Mozilla Public License 2.0
pragma solidity ^0.8.17;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { Context } from "@openzeppelin/contracts/utils/Context.sol";
import { CommunityOwnable } from "../CommunityOwnable.sol";

contract CommunityERC20 is Context, Ownable, ERC20, CommunityOwnable {
    error CommunityERC20_MaxSupplyLowerThanTotalSupply();
    error CommunityERC20_MaxSupplyReached();
    error CommunityERC20_MismatchingAddressesAndAmountsLengths();

    /// @notice Emits a custom mint event for Status applications to listen to
    /// @dev This is doubling the {Transfer} event from ERC20 but we need to emit this
    /// so Status applications have a way to easily distinguish between transactions that have
    /// a similar event footprint but are semantically different.
    /// @param from The address that minted the token
    /// @param to The address that received the token
    /// @param amount The amount that was minted
    event StatusMint(address indexed from, address indexed to, uint256 indexed amount);

    /**
     * If we want unlimited total supply we should set maxSupply to 2^256-1.
     */
    uint256 public maxSupply;

    uint8 private immutable customDecimals;

    string public baseTokenURI;

    constructor(
        string memory _name,
        string memory _symbol,
        uint8 _decimals,
        uint256 _maxSupply,
        string memory _baseTokenURI,
        address _ownerToken,
        address _masterToken
    )
        ERC20(_name, _symbol)
        CommunityOwnable(_ownerToken, _masterToken)
    {
        maxSupply = _maxSupply;
        customDecimals = _decimals;
        baseTokenURI = _baseTokenURI;
    }

    // Events

    // External functions

    function setMaxSupply(uint256 newMaxSupply) external onlyCommunityOwnerOrTokenMaster {
        if (newMaxSupply < totalSupply()) {
            revert CommunityERC20_MaxSupplyLowerThanTotalSupply();
        }
        maxSupply = newMaxSupply;
    }

    /**
     * @dev Mint tokens for each address in `addresses` each one with
     * an amount specified in `amounts`.
     *
     */
    function mintTo(address[] memory addresses, uint256[] memory amounts) external onlyCommunityOwnerOrTokenMaster {
        if (addresses.length != amounts.length) {
            revert CommunityERC20_MismatchingAddressesAndAmountsLengths();
        }

        for (uint256 i = 0; i < addresses.length; i++) {
            uint256 amount = amounts[i];
            if (totalSupply() + amount > maxSupply) {
                revert CommunityERC20_MaxSupplyReached();
            }
            _mint(addresses[i], amount);
            emit StatusMint(address(0), addresses[i], amount);
        }
    }

    // Public functions
    function decimals() public view virtual override returns (uint8) {
        return customDecimals;
    }

    // Internal functions

    // Private functions
}
