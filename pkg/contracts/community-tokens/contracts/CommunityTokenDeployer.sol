// SPDX-License-Identifier: UNLICENSED

pragma solidity ^0.8.17;

import { Ownable2Step } from "@openzeppelin/contracts/access/Ownable2Step.sol";
import { ECDSA } from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import { EIP712 } from "@openzeppelin/contracts/utils/cryptography/EIP712.sol";
import { ITokenFactory } from "./interfaces/ITokenFactory.sol";
import { IAddressRegistry } from "./interfaces/IAddressRegistry.sol";

/**
 * @title CommunityTokenDeployer contract
 * @author 0x-r4bbit
 *
 * This contract serves as a deployment process for Status community owners
 * to deploy access control token contracts on behalf of their Status community.
 * The contract keep a reference to token factories that are used for deploying tokens.
 * The contract deploys the two token contracts `OwnerToken` and `MasterToken` those factories.
 * The contract maintains a registry which keeps track of `OwnerToken` contract
 * addresses per community.
 *
 * Only one deployment per community can be done.
 * Status community owners have to provide an EIP712 hash signature that was
 * created using their community's private key to successfully execute a deployment.
 *
 * @notice This contract is used by Status community owners to deploy
 * community access control token contracts.
 * @notice This contract maintains a registry that tracks contract addresses
 * and community addresses
 * @dev This contract will be deployed by Status, making Status the owner
 * of the contract.
 * @dev A contract address for a `CommunityTokenRegistry` contract has to be provided
 * to create this contract.
 * @dev A contract address for a `CommunityOwnerTokenFactory` contract has to be provided
 * to create this contract.
 * @dev A contract address for a `CommunityMasterTokenFactory` contract has to be provided
 * to create this contract.
 * @dev The `CommunityTokenRegistry` address can be changed by the owner of this contract.
 * @dev The `CommunityOwnerTokenFactory` address can be changed by the owner of this contract.
 * @dev The `CommunityMasterTokenFactory` address can be changed by the owner of this contract.
 */
contract CommunityTokenDeployer is EIP712("CommunityTokenDeployer", "1"), Ownable2Step {
    using ECDSA for bytes32;

    error CommunityTokenDeployer_InvalidDeploymentRegistryAddress();
    error CommunityTokenDeployer_InvalidTokenFactoryAddress();
    error CommunityTokenDeployer_EqualFactoryAddresses();
    error CommunityTokenDeployer_AlreadyDeployed();
    error CommunityTokenDeployer_InvalidSignerKeyOrCommunityAddress();
    error CommunityTokenDeployer_InvalidTokenMetadata();
    error CommunityTokenDeployer_InvalidDeployerAddress();
    error CommunityTokenDeployer_InvalidDeploymentSignature();

    event OwnerTokenFactoryAddressChange(address indexed);
    event MasterTokenFactoryAddressChange(address indexed);
    event DeploymentRegistryAddressChange(address indexed);
    event DeployOwnerToken(address indexed);
    event DeployMasterToken(address indexed);

    /// @dev Needed to avoid "Stack too deep" error.
    struct TokenConfig {
        string name;
        string symbol;
        string baseURI;
    }

    /// @dev Used to verify signatures.
    struct DeploymentSignature {
        address signer;
        address deployer;
        uint8 v;
        bytes32 r;
        bytes32 s;
    }

    bytes32 public constant DEPLOYMENT_SIGNATURE_TYPEHASH = keccak256("Deploy(address signer,address deployer)");

    /// @dev Address of the `CommunityTokenRegistry` contract instance.
    address public deploymentRegistry;

    /// @dev Address of the `CommunityOwnerTokenFactory` contract instance.
    address public ownerTokenFactory;

    /// @dev Address of the `CommunityMasterTokenFactory` contract instance.
    address public masterTokenFactory;

    /// @param _registry The address of the `CommunityTokenRegistry` contract.
    /// @param _ownerTokenFactory The address of the `CommunityOwnerTokenFactory` contract.
    /// @param _masterTokenFactory The address of the `CommunityMasterTokenFactory` contract.
    constructor(address _registry, address _ownerTokenFactory, address _masterTokenFactory) {
        if (_registry == address(0)) {
            revert CommunityTokenDeployer_InvalidDeploymentRegistryAddress();
        }
        if (_ownerTokenFactory == address(0) || _masterTokenFactory == address(0)) {
            revert CommunityTokenDeployer_InvalidTokenFactoryAddress();
        }
        if (_ownerTokenFactory == _masterTokenFactory) {
            revert CommunityTokenDeployer_EqualFactoryAddresses();
        }
        deploymentRegistry = _registry;
        ownerTokenFactory = _ownerTokenFactory;
        masterTokenFactory = _masterTokenFactory;
    }

    /**
     * @notice Deploys an instance of `OwnerToken` and `MasterToken` on behalf
     * of a Status community account, provided `_signature` is valid and was signed
     * by that Status community account, using the configured factory contracts.
     * @dev Anyone can call this function but a valid EIP712 hash signature has to be
     * provided for a successful deployment.
     * @dev Emits {CreateToken} events via underlying token factories.
     * @dev Emits {DeployOwnerToken} event.
     * @dev Emits {DeployMasterToken} event.
     *
     * @param _ownerToken A `TokenConfig` containing ERC721 metadata for `OwnerToken`
     * @param _masterToken A `TokenConfig` containing ERC721 metadata for `MasterToken`
     * @param _signature A `DeploymentSignature` containing a signer and deployer address,
     * and a signature created by a Status community
     * @return address The address of the deployed `OwnerToken` contract.
     * @return address The address of the deployed `MasterToken` contract.
     */
    function deploy(
        TokenConfig calldata _ownerToken,
        TokenConfig calldata _masterToken,
        DeploymentSignature calldata _signature,
        bytes memory _signerPublicKey
    )
        external
        returns (address, address)
    {
        if (_signature.signer == address(0) || _signerPublicKey.length == 0) {
            revert CommunityTokenDeployer_InvalidSignerKeyOrCommunityAddress();
        }

        if (_signature.deployer != msg.sender) {
            revert CommunityTokenDeployer_InvalidDeployerAddress();
        }

        if (IAddressRegistry(deploymentRegistry).getEntry(_signature.signer) != address(0)) {
            revert CommunityTokenDeployer_AlreadyDeployed();
        }

        if (!_verifySignature(_signature)) {
            revert CommunityTokenDeployer_InvalidDeploymentSignature();
        }

        address ownerToken = ITokenFactory(ownerTokenFactory).create(
            _ownerToken.name, _ownerToken.symbol, _ownerToken.baseURI, msg.sender, _signerPublicKey
        );

        emit DeployOwnerToken(ownerToken);

        address masterToken = ITokenFactory(masterTokenFactory).create(
            _masterToken.name, _masterToken.symbol, _masterToken.baseURI, ownerToken, bytes("")
        );

        emit DeployMasterToken(masterToken);

        IAddressRegistry(deploymentRegistry).addEntry(_signature.signer, ownerToken);
        return (ownerToken, masterToken);
    }

    /**
     * @notice Sets a deployment registry address.
     * @dev Only the owner can call this function.
     * @dev Emits a {DeploymentRegistryAddressChange} event.
     * @dev Reverts if the provided address is a zero address.
     *
     * @param _deploymentRegistry The address of the deployment registry contract.
     */
    function setDeploymentRegistryAddress(address _deploymentRegistry) external onlyOwner {
        if (_deploymentRegistry == address(0)) {
            revert CommunityTokenDeployer_InvalidDeploymentRegistryAddress();
        }
        deploymentRegistry = _deploymentRegistry;
        emit DeploymentRegistryAddressChange(deploymentRegistry);
    }

    /**
     * @notice Sets the `OwnerToken` factory contract address.
     * @dev Only the owner can call this function.
     * @dev Emits a {OwnerTokenFactoryChange} event.
     * @dev Reverts if the provided address is a zero address.
     *
     * @param _ownerTokenFactory The address of the `OwnerToken` factory contract.
     */
    function setOwnerTokenFactoryAddress(address _ownerTokenFactory) external onlyOwner {
        if (_ownerTokenFactory == address(0)) {
            revert CommunityTokenDeployer_InvalidTokenFactoryAddress();
        }
        ownerTokenFactory = _ownerTokenFactory;
        emit OwnerTokenFactoryAddressChange(ownerTokenFactory);
    }

    /**
     * @notice Sets the `MasterToken` factory contract address.
     * @dev Only the owner can call this function.
     * @dev Emits a {MasterTokenFactoryChange} event.
     * @dev Reverts if the provided address is a zero address.
     *
     * @param _masterTokenFactory The address of the `MasterToken` factory contract.
     */
    function setMasterTokenFactoryAddress(address _masterTokenFactory) external onlyOwner {
        if (_masterTokenFactory == address(0)) {
            revert CommunityTokenDeployer_InvalidTokenFactoryAddress();
        }
        masterTokenFactory = _masterTokenFactory;
        emit MasterTokenFactoryAddressChange(masterTokenFactory);
    }

    /**
     * @notice Returns an EIP712 domain separator hash
     * @return bytes32 An EIP712 domain separator hash
     */
    function DOMAIN_SEPARATOR() external view returns (bytes32) {
        return _domainSeparatorV4();
    }

    /**
     * @notice Verifies provided `DeploymentSignature` which was created by
     * the Status community account for which the access control token contracts
     * will be deployed.
     * @dev This contract does not maintain nonces for the typed data hash, which
     * is typically done to prevent signature replay attacks. The `deploy()` function
     * allows only one deployment per Status community, so replay attacks are not possible.
     * @return bool Whether the provided signature could be recovered.
     */
    function _verifySignature(DeploymentSignature calldata signature) internal view returns (bool) {
        bytes32 digest =
            _hashTypedDataV4(keccak256(abi.encode(DEPLOYMENT_SIGNATURE_TYPEHASH, signature.signer, signature.deployer)));
        return signature.signer == digest.recover(signature.v, signature.r, signature.s);
    }
}
