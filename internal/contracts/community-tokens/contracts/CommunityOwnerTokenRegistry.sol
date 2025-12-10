// SPDX-License-Identifier: UNLICENSED

pragma solidity ^0.8.17;

import { Ownable2Step } from "@openzeppelin/contracts/access/Ownable2Step.sol";
import { IAddressRegistry } from "./interfaces/IAddressRegistry.sol";
import { OwnerToken } from "./tokens/OwnerToken.sol";

/**
 * @title CommunityOwnerTokenRegistry contract
 * @author 0x-r4bbit
 *
 * This contract serves as a simple registry to map Status community addresses
 * to Status community `OwnerToken` addresses.
 * The `CommunityTokenDeployer` contract uses this registry contract to maintain
 * a list of community address and their token addresses.
 * @notice This contract will be deployed by Status similar to the `CommunityTokenDeployer`
 * contract.
 * @notice This contract maps community addresses to `OwnerToken` addresses.
 * @notice Only one entry per community address can exist in the registry.
 * @dev This registry has been extracted into its own contract so that it's possible
 * to introduce different version of a `CommunityDeployerContract` without needing to
 * migrate existing registry data, as the deployer contract would simply point at this
 * registry contract.
 * @dev Only `tokenDeployer` can add entries to the registry.
 */
contract CommunityOwnerTokenRegistry is IAddressRegistry, Ownable2Step {
    error CommunityOwnerTokenRegistry_NotAuthorized();
    error CommunityOwnerTokenRegistry_EntryAlreadyExists();
    error CommunityOwnerTokenRegistry_InvalidAddress();

    event TokenDeployerAddressChange(address indexed);
    event AddEntry(address indexed, address indexed);

    /// @dev The address of the token deployer contract.
    address public tokenDeployer;

    mapping(address => address) public communityAddressToTokenAddress;

    modifier onlyTokenDeployer() {
        if (msg.sender != tokenDeployer) {
            revert CommunityOwnerTokenRegistry_NotAuthorized();
        }
        _;
    }

    /**
     * @notice Sets the address of the community token deployer contract. This is needed to
     * ensure only the known token deployer contract can add new entries to the registry.
     * @dev Only the owner of this contract can call this function.
     * @dev Emits a {TokenDeployerAddressChange} event.
     *
     * @param _tokenDeployer The address of the community token deployer contract
     */
    function setCommunityTokenDeployerAddress(address _tokenDeployer) external onlyOwner {
        if (_tokenDeployer == address(0)) {
            revert CommunityOwnerTokenRegistry_InvalidAddress();
        }
        tokenDeployer = _tokenDeployer;
        emit TokenDeployerAddressChange(tokenDeployer);
    }

    /**
     * @notice Adds an entry to the registry. Only one entry per community address can exist.
     * @dev Only the token deployer contract can call this function.
     * @dev Reverts when the entry already exists.
     * @dev Reverts when either `_communityAddress` or `_tokenAddress` are zero addresses.
     * @dev Emits a {AddEntry} event.
     */
    function addEntry(address _communityAddress, address _tokenAddress) external onlyTokenDeployer {
        if (getEntry(_communityAddress) != address(0)) {
            revert CommunityOwnerTokenRegistry_EntryAlreadyExists();
        }

        if (_communityAddress == address(0) || _tokenAddress == address(0)) {
            revert CommunityOwnerTokenRegistry_InvalidAddress();
        }

        communityAddressToTokenAddress[_communityAddress] = _tokenAddress;
        emit AddEntry(_communityAddress, _tokenAddress);
    }

    /**
     * @notice Returns the owner token address for a given community address.
     * @param _communityAddress The community address to look up an owner token address.
     * @return address The owner token address for the community addres, or zero address .
     */
    function getEntry(address _communityAddress) public view returns (address) {
        return communityAddressToTokenAddress[_communityAddress];
    }
}
