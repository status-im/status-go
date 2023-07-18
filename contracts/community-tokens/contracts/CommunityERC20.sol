// SPDX-License-Identifier: Mozilla Public License 2.0
pragma solidity ^0.8.17;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/utils/Context.sol";

contract CommunityERC20 is
    Context,
    Ownable,
    ERC20
{
    /**
     * If we want unlimited total supply we should set maxSupply to 2^256-1.
     */
    uint256 public maxSupply;

    uint8 private customDecimals;

    constructor(
        string memory _name,
        string memory _symbol,
        uint8 _decimals,
        uint256 _maxSupply
    ) ERC20(_name, _symbol) {
        maxSupply = _maxSupply;
        customDecimals = _decimals;
    }

    // Events

    // External functions

    function setMaxSupply(uint256 newMaxSupply) external onlyOwner {
        require(newMaxSupply >= totalSupply(), "MAX_SUPPLY_LOWER_THAN_TOTAL_SUPPLY");
        maxSupply = newMaxSupply;
    }

    /**
     * @dev Mint tokens for each address in `addresses` each one with
     * an amount specified in `amounts`.
     *
     */
    function mintTo(address[] memory addresses, uint256[] memory amounts) external onlyOwner {
        require(addresses.length == amounts.length, "WRONG_LENGTHS");

        for (uint256 i = 0; i < addresses.length; i++) {
            uint256 amount = amounts[i];
            require(totalSupply() + amount <= maxSupply, "MAX_SUPPLY_REACHED");
            _mint(addresses[i], amount);
        }
    }

    // Public functions
    function decimals() public view virtual override returns (uint8) {
        return customDecimals;
    }

    // Internal functions

    // Private functions
}
