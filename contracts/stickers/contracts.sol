pragma solidity ^0.5.0;

/**
 * @dev Wrappers over Solidity's arithmetic operations with added overflow
 * checks.
 *
 * Arithmetic operations in Solidity wrap on overflow. This can easily result
 * in bugs, because programmers usually assume that an overflow raises an
 * error, which is the standard behavior in high level programming languages.
 * `SafeMath` restores this intuition by reverting the transaction when an
 * operation overflows.
 *
 * Using this library instead of the unchecked operations eliminates an entire
 * class of bugs, so it's recommended to use it always.
 */
library SafeMath {
    /**
     * @dev Returns the addition of two unsigned integers, reverting on
     * overflow.
     *
     * Counterpart to Solidity's `+` operator.
     *
     * Requirements:
     * - Addition cannot overflow.
     */
    function add(uint256 a, uint256 b) internal pure returns (uint256) {
        uint256 c = a + b;
        require(c >= a, "SafeMath: addition overflow");

        return c;
    }

    /**
     * @dev Returns the subtraction of two unsigned integers, reverting on
     * overflow (when the result is negative).
     *
     * Counterpart to Solidity's `-` operator.
     *
     * Requirements:
     * - Subtraction cannot overflow.
     */
    function sub(uint256 a, uint256 b) internal pure returns (uint256) {
        require(b <= a, "SafeMath: subtraction overflow");
        uint256 c = a - b;

        return c;
    }

    /**
     * @dev Returns the multiplication of two unsigned integers, reverting on
     * overflow.
     *
     * Counterpart to Solidity's `*` operator.
     *
     * Requirements:
     * - Multiplication cannot overflow.
     */
    function mul(uint256 a, uint256 b) internal pure returns (uint256) {
        // Gas optimization: this is cheaper than requiring 'a' not being zero, but the
        // benefit is lost if 'b' is also tested.
        // See: https://github.com/OpenZeppelin/openzeppelin-solidity/pull/522
        if (a == 0) {
            return 0;
        }

        uint256 c = a * b;
        require(c / a == b, "SafeMath: multiplication overflow");

        return c;
    }

    /**
     * @dev Returns the integer division of two unsigned integers. Reverts on
     * division by zero. The result is rounded towards zero.
     *
     * Counterpart to Solidity's `/` operator. Note: this function uses a
     * `revert` opcode (which leaves remaining gas untouched) while Solidity
     * uses an invalid opcode to revert (consuming all remaining gas).
     *
     * Requirements:
     * - The divisor cannot be zero.
     */
    function div(uint256 a, uint256 b) internal pure returns (uint256) {
        // Solidity only automatically asserts when dividing by 0
        require(b > 0, "SafeMath: division by zero");
        uint256 c = a / b;
        // assert(a == b * c + a % b); // There is no case in which this doesn't hold

        return c;
    }

    /**
     * @dev Returns the remainder of dividing two unsigned integers. (unsigned integer modulo),
     * Reverts when dividing by zero.
     *
     * Counterpart to Solidity's `%` operator. This function uses a `revert`
     * opcode (which leaves remaining gas untouched) while Solidity uses an
     * invalid opcode to revert (consuming all remaining gas).
     *
     * Requirements:
     * - The divisor cannot be zero.
     */
    function mod(uint256 a, uint256 b) internal pure returns (uint256) {
        require(b != 0, "SafeMath: modulo by zero");
        return a % b;
    }
}


/**
 * @dev Collection of functions related to the address type,
 */
library Address {
    /**
     * @dev Returns true if `account` is a contract.
     *
     * This test is non-exhaustive, and there may be false-negatives: during the
     * execution of a contract's constructor, its address will be reported as
     * not containing a contract.
     *
     * > It is unsafe to assume that an address for which this function returns
     * false is an externally-owned account (EOA) and not a contract.
     */
    function isContract(address account) internal view returns (bool) {
        // This method relies in extcodesize, which returns 0 for contracts in
        // construction, since the code is only stored at the end of the
        // constructor execution.

        uint256 size;
        // solhint-disable-next-line no-inline-assembly
        assembly { size := extcodesize(account) }
        return size > 0;
    }
}


contract Controlled {
    event NewController(address controller);
    /// @notice The address of the controller is the only address that can call
    ///  a function with this modifier
    modifier onlyController {
        require(msg.sender == controller, "Unauthorized");
        _;
    }

    address payable public controller;

    constructor() internal {
        controller = msg.sender;
    }

    /// @notice Changes the controller of the contract
    /// @param _newController The new controller of the contract
    function changeController(address payable _newController) public onlyController {
        controller = _newController;
        emit NewController(_newController);
    }
}


// Abstract contract for the full ERC 20 Token standard
// https://github.com/ethereum/EIPs/issues/20

interface ERC20Token {

    /**
     * @notice send `_value` token to `_to` from `msg.sender`
     * @param _to The address of the recipient
     * @param _value The amount of token to be transferred
     * @return Whether the transfer was successful or not
     */
    function transfer(address _to, uint256 _value) external returns (bool success);

    /**
     * @notice `msg.sender` approves `_spender` to spend `_value` tokens
     * @param _spender The address of the account able to transfer the tokens
     * @param _value The amount of tokens to be approved for transfer
     * @return Whether the approval was successful or not
     */
    function approve(address _spender, uint256 _value) external returns (bool success);

    /**
     * @notice send `_value` token to `_to` from `_from` on the condition it is approved by `_from`
     * @param _from The address of the sender
     * @param _to The address of the recipient
     * @param _value The amount of token to be transferred
     * @return Whether the transfer was successful or not
     */
    function transferFrom(address _from, address _to, uint256 _value) external returns (bool success);

    /**
     * @param _owner The address from which the balance will be retrieved
     * @return The balance
     */
    function balanceOf(address _owner) external view returns (uint256 balance);

    /**
     * @param _owner The address of the account owning tokens
     * @param _spender The address of the account able to transfer the tokens
     * @return Amount of remaining tokens allowed to spent
     */
    function allowance(address _owner, address _spender) external view returns (uint256 remaining);

    /**
     * @notice return total supply of tokens
     */
    function totalSupply() external view returns (uint256 supply);

    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
}


/**
 * @dev Interface of the ERC165 standard, as defined in the
 * [EIP](https://eips.ethereum.org/EIPS/eip-165).
 *
 * Implementers can declare support of contract interfaces, which can then be
 * queried by others (`ERC165Checker`).
 *
 * For an implementation, see `ERC165`.
 */
interface IERC165 {
    /**
     * @dev Returns true if this contract implements the interface defined by
     * `interfaceId`. See the corresponding
     * [EIP section](https://eips.ethereum.org/EIPS/eip-165#how-interfaces-are-identified)
     * to learn more about how these ids are created.
     *
     * This function call must use less than 30 000 gas.
     */
    function supportsInterface(bytes4 interfaceId) external view returns (bool);
}


/**
 * @title ERC721 token receiver interface
 * @dev Interface for any contract that wants to support safeTransfers
 * from ERC721 asset contracts.
 */
contract IERC721Receiver {
    /**
     * @notice Handle the receipt of an NFT
     * @dev The ERC721 smart contract calls this function on the recipient
     * after a `safeTransfer`. This function MUST return the function selector,
     * otherwise the caller will revert the transaction. The selector to be
     * returned can be obtained as `this.onERC721Received.selector`. This
     * function MAY throw to revert and reject the transfer.
     * Note: the ERC721 contract address is always the message sender.
     * @param operator The address which called `safeTransferFrom` function
     * @param from The address which previously owned the token
     * @param tokenId The NFT identifier which is being transferred
     * @param data Additional data with no specified format
     * @return bytes4 `bytes4(keccak256("onERC721Received(address,address,uint256,bytes)"))`
     */
    function onERC721Received(address operator, address from, uint256 tokenId, bytes memory data)
        public returns (bytes4);
}


/**
 * @title Counters
 * @author Matt Condon (@shrugs)
 * @dev Provides counters that can only be incremented or decremented by one. This can be used e.g. to track the number
 * of elements in a mapping, issuing ERC721 ids, or counting request ids.
 *
 * Include with `using Counters for Counters.Counter;`
 * Since it is not possible to overflow a 256 bit integer with increments of one, `increment` can skip the SafeMath
 * overflow check, thereby saving gas. This does assume however correct usage, in that the underlying `_value` is never
 * directly accessed.
 */
library Counters {
    using SafeMath for uint256;

    struct Counter {
        // This variable should never be directly accessed by users of the library: interactions must be restricted to
        // the library's function. As of Solidity v0.5.2, this cannot be enforced, though there is a proposal to add
        // this feature: see https://github.com/ethereum/solidity/issues/4637
        uint256 _value; // default: 0
    }

    function current(Counter storage counter) internal view returns (uint256) {
        return counter._value;
    }

    function increment(Counter storage counter) internal {
        counter._value += 1;
    }

    function decrement(Counter storage counter) internal {
        counter._value = counter._value.sub(1);
    }
}


/**
 * @dev Implementation of the `IERC165` interface.
 *
 * Contracts may inherit from this and call `_registerInterface` to declare
 * their support of an interface.
 */
contract ERC165 is IERC165 {
    /*
     * bytes4(keccak256('supportsInterface(bytes4)')) == 0x01ffc9a7
     */
    bytes4 private constant _INTERFACE_ID_ERC165 = 0x01ffc9a7;

    /**
     * @dev Mapping of interface ids to whether or not it's supported.
     */
    mapping(bytes4 => bool) private _supportedInterfaces;

    constructor () internal {
        // Derived contracts need only register support for their own interfaces,
        // we register support for ERC165 itself here
        _registerInterface(_INTERFACE_ID_ERC165);
    }

    /**
     * @dev See `IERC165.supportsInterface`.
     *
     * Time complexity O(1), guaranteed to always use less than 30 000 gas.
     */
    function supportsInterface(bytes4 interfaceId) external view returns (bool) {
        return _supportedInterfaces[interfaceId];
    }

    /**
     * @dev Registers the contract as an implementer of the interface defined by
     * `interfaceId`. Support of the actual ERC165 interface is automatic and
     * registering its interface id is not required.
     *
     * See `IERC165.supportsInterface`.
     *
     * Requirements:
     *
     * - `interfaceId` cannot be the ERC165 invalid interface (`0xffffffff`).
     */
    function _registerInterface(bytes4 interfaceId) internal {
        require(interfaceId != 0xffffffff, "ERC165: invalid interface id");
        _supportedInterfaces[interfaceId] = true;
    }
}


contract ERC20Receiver {

    event TokenDeposited(address indexed token, address indexed sender, uint256 amount);
    event TokenWithdrawn(address indexed token, address indexed sender, uint256 amount);

    mapping (address => mapping(address => uint256)) tokenBalances;

    constructor() public {
    }

    function depositToken(
        ERC20Token _token
    )
        external
    {
        _depositToken(
            msg.sender,
            _token,
            _token.allowance(
                msg.sender,
                address(this)
            )
        );
    }

    function withdrawToken(
        ERC20Token _token,
        uint256 _amount
    )
        external
    {
        _withdrawToken(msg.sender, _token, _amount);
    }

    function depositToken(
        ERC20Token _token,
        uint256 _amount
    )
        external
    {
        require(_token.allowance(msg.sender, address(this)) >= _amount, "Bad argument");
        _depositToken(msg.sender, _token, _amount);
    }

    function tokenBalanceOf(
        ERC20Token _token,
        address _from
    )
        external
        view
        returns(uint256 fromTokenBalance)
    {
        return tokenBalances[address(_token)][_from];
    }

    function _depositToken(
        address _from,
        ERC20Token _token,
        uint256 _amount
    )
        private
    {
        require(_amount > 0, "Bad argument");
        if (_token.transferFrom(_from, address(this), _amount)) {
            tokenBalances[address(_token)][_from] += _amount;
            emit TokenDeposited(address(_token), _from, _amount);
        }
    }

    function _withdrawToken(
        address _from,
        ERC20Token _token,
        uint256 _amount
    )
        private
    {
        require(_amount > 0, "Bad argument");
        require(tokenBalances[address(_token)][_from] >= _amount, "Insufficient funds");
        tokenBalances[address(_token)][_from] -= _amount;
        require(_token.transfer(_from, _amount), "Transfer fail");
        emit TokenWithdrawn(address(_token), _from, _amount);
    }

}


/**
 * @dev Required interface of an ERC721 compliant contract.
 */
contract IERC721 is IERC165 {
    event Transfer(address indexed from, address indexed to, uint256 indexed tokenId);
    event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId);
    event ApprovalForAll(address indexed owner, address indexed operator, bool approved);

    /**
     * @dev Returns the number of NFTs in `owner`'s account.
     */
    function balanceOf(address owner) public view returns (uint256 balance);

    /**
     * @dev Returns the owner of the NFT specified by `tokenId`.
     */
    function ownerOf(uint256 tokenId) public view returns (address owner);

    /**
     * @dev Transfers a specific NFT (`tokenId`) from one account (`from`) to
     * another (`to`).
     *
     * 
     *
     * Requirements:
     * - `from`, `to` cannot be zero.
     * - `tokenId` must be owned by `from`.
     * - If the caller is not `from`, it must be have been allowed to move this
     * NFT by either `approve` or `setApproveForAll`.
     */
    function safeTransferFrom(address from, address to, uint256 tokenId) public;
    /**
     * @dev Transfers a specific NFT (`tokenId`) from one account (`from`) to
     * another (`to`).
     *
     * Requirements:
     * - If the caller is not `from`, it must be approved to move this NFT by
     * either `approve` or `setApproveForAll`.
     */
    function transferFrom(address from, address to, uint256 tokenId) public;
    function approve(address to, uint256 tokenId) public;
    function getApproved(uint256 tokenId) public view returns (address operator);

    function setApprovalForAll(address operator, bool _approved) public;
    function isApprovedForAll(address owner, address operator) public view returns (bool);


    function safeTransferFrom(address from, address to, uint256 tokenId, bytes memory data) public;
}


/**
 * @title ERC-721 Non-Fungible Token Standard, optional enumeration extension
 * @dev See https://eips.ethereum.org/EIPS/eip-721
 */
contract IERC721Enumerable is IERC721 {
    function totalSupply() public view returns (uint256);
    function tokenOfOwnerByIndex(address owner, uint256 index) public view returns (uint256 tokenId);

    function tokenByIndex(uint256 index) public view returns (uint256);
}


/**
 * @title ERC-721 Non-Fungible Token Standard, optional metadata extension
 * @dev See https://eips.ethereum.org/EIPS/eip-721
 */
contract IERC721Metadata is IERC721 {
    function name() external view returns (string memory);
    function symbol() external view returns (string memory);
    function tokenURI(uint256 tokenId) external view returns (string memory);
}


contract TokenClaimer {
    event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount);

    function claimTokens(address _token) external;
    /**
     * @notice This method can be used by the controller to extract mistakenly
     *  sent tokens to this contract.
     * @param _token The address of the token contract that you want to recover
     *  set to 0 in case you want to extract ether.
     */
    function withdrawBalance(address _token, address payable _destination)
        internal
    {
        uint256 balance;
        if (_token == address(0)) {
            balance = address(this).balance;
            address(_destination).transfer(balance);
        } else {
            ERC20Token token = ERC20Token(_token);
            balance = token.balanceOf(address(this));
            token.transfer(_destination, balance);
        }
        emit ClaimedTokens(_token, _destination, balance);
    }
}


/**
 * @title ERC721 Non-Fungible Token Standard basic implementation
 * @dev see https://eips.ethereum.org/EIPS/eip-721
 */
contract ERC721 is ERC165, IERC721 {
    using SafeMath for uint256;
    using Address for address;
    using Counters for Counters.Counter;

    // Equals to `bytes4(keccak256("onERC721Received(address,address,uint256,bytes)"))`
    // which can be also obtained as `IERC721Receiver(0).onERC721Received.selector`
    bytes4 private constant _ERC721_RECEIVED = 0x150b7a02;

    // Mapping from token ID to owner
    mapping (uint256 => address) private _tokenOwner;

    // Mapping from token ID to approved address
    mapping (uint256 => address) private _tokenApprovals;

    // Mapping from owner to number of owned token
    mapping (address => Counters.Counter) private _ownedTokensCount;

    // Mapping from owner to operator approvals
    mapping (address => mapping (address => bool)) private _operatorApprovals;

    /*
     *     bytes4(keccak256('balanceOf(address)')) == 0x70a08231
     *     bytes4(keccak256('ownerOf(uint256)')) == 0x6352211e
     *     bytes4(keccak256('approve(address,uint256)')) == 0x095ea7b3
     *     bytes4(keccak256('getApproved(uint256)')) == 0x081812fc
     *     bytes4(keccak256('setApprovalForAll(address,bool)')) == 0xa22cb465
     *     bytes4(keccak256('isApprovedForAll(address,address)')) == 0xe985e9c
     *     bytes4(keccak256('transferFrom(address,address,uint256)')) == 0x23b872dd
     *     bytes4(keccak256('safeTransferFrom(address,address,uint256)')) == 0x42842e0e
     *     bytes4(keccak256('safeTransferFrom(address,address,uint256,bytes)')) == 0xb88d4fde
     *
     *     => 0x70a08231 ^ 0x6352211e ^ 0x095ea7b3 ^ 0x081812fc ^
     *        0xa22cb465 ^ 0xe985e9c ^ 0x23b872dd ^ 0x42842e0e ^ 0xb88d4fde == 0x80ac58cd
     */
    bytes4 private constant _INTERFACE_ID_ERC721 = 0x80ac58cd;

    constructor () public {
        // register the supported interfaces to conform to ERC721 via ERC165
        _registerInterface(_INTERFACE_ID_ERC721);
    }

    /**
     * @dev Gets the balance of the specified address.
     * @param owner address to query the balance of
     * @return uint256 representing the amount owned by the passed address
     */
    function balanceOf(address owner) public view returns (uint256) {
        require(owner != address(0), "ERC721: balance query for the zero address");

        return _ownedTokensCount[owner].current();
    }

    /**
     * @dev Gets the owner of the specified token ID.
     * @param tokenId uint256 ID of the token to query the owner of
     * @return address currently marked as the owner of the given token ID
     */
    function ownerOf(uint256 tokenId) public view returns (address) {
        address owner = _tokenOwner[tokenId];
        require(owner != address(0), "ERC721: owner query for nonexistent token");

        return owner;
    }

    /**
     * @dev Approves another address to transfer the given token ID
     * The zero address indicates there is no approved address.
     * There can only be one approved address per token at a given time.
     * Can only be called by the token owner or an approved operator.
     * @param to address to be approved for the given token ID
     * @param tokenId uint256 ID of the token to be approved
     */
    function approve(address to, uint256 tokenId) public {
        address owner = ownerOf(tokenId);
        require(to != owner, "ERC721: approval to current owner");

        require(msg.sender == owner || isApprovedForAll(owner, msg.sender),
            "ERC721: approve caller is not owner nor approved for all"
        );

        _tokenApprovals[tokenId] = to;
        emit Approval(owner, to, tokenId);
    }

    /**
     * @dev Gets the approved address for a token ID, or zero if no address set
     * Reverts if the token ID does not exist.
     * @param tokenId uint256 ID of the token to query the approval of
     * @return address currently approved for the given token ID
     */
    function getApproved(uint256 tokenId) public view returns (address) {
        require(_exists(tokenId), "ERC721: approved query for nonexistent token");

        return _tokenApprovals[tokenId];
    }

    /**
     * @dev Sets or unsets the approval of a given operator
     * An operator is allowed to transfer all tokens of the sender on their behalf.
     * @param to operator address to set the approval
     * @param approved representing the status of the approval to be set
     */
    function setApprovalForAll(address to, bool approved) public {
        require(to != msg.sender, "ERC721: approve to caller");

        _operatorApprovals[msg.sender][to] = approved;
        emit ApprovalForAll(msg.sender, to, approved);
    }

    /**
     * @dev Tells whether an operator is approved by a given owner.
     * @param owner owner address which you want to query the approval of
     * @param operator operator address which you want to query the approval of
     * @return bool whether the given operator is approved by the given owner
     */
    function isApprovedForAll(address owner, address operator) public view returns (bool) {
        return _operatorApprovals[owner][operator];
    }

    /**
     * @dev Transfers the ownership of a given token ID to another address.
     * Usage of this method is discouraged, use `safeTransferFrom` whenever possible.
     * Requires the msg.sender to be the owner, approved, or operator.
     * @param from current owner of the token
     * @param to address to receive the ownership of the given token ID
     * @param tokenId uint256 ID of the token to be transferred
     */
    function transferFrom(address from, address to, uint256 tokenId) public {
        //solhint-disable-next-line max-line-length
        require(_isApprovedOrOwner(msg.sender, tokenId), "ERC721: transfer caller is not owner nor approved");

        _transferFrom(from, to, tokenId);
    }

    /**
     * @dev Safely transfers the ownership of a given token ID to another address
     * If the target address is a contract, it must implement `onERC721Received`,
     * which is called upon a safe transfer, and return the magic value
     * `bytes4(keccak256("onERC721Received(address,address,uint256,bytes)"))`; otherwise,
     * the transfer is reverted.
     * Requires the msg.sender to be the owner, approved, or operator
     * @param from current owner of the token
     * @param to address to receive the ownership of the given token ID
     * @param tokenId uint256 ID of the token to be transferred
     */
    function safeTransferFrom(address from, address to, uint256 tokenId) public {
        safeTransferFrom(from, to, tokenId, "");
    }

    /**
     * @dev Safely transfers the ownership of a given token ID to another address
     * If the target address is a contract, it must implement `onERC721Received`,
     * which is called upon a safe transfer, and return the magic value
     * `bytes4(keccak256("onERC721Received(address,address,uint256,bytes)"))`; otherwise,
     * the transfer is reverted.
     * Requires the msg.sender to be the owner, approved, or operator
     * @param from current owner of the token
     * @param to address to receive the ownership of the given token ID
     * @param tokenId uint256 ID of the token to be transferred
     * @param _data bytes data to send along with a safe transfer check
     */
    function safeTransferFrom(address from, address to, uint256 tokenId, bytes memory _data) public {
        transferFrom(from, to, tokenId);
        require(_checkOnERC721Received(from, to, tokenId, _data), "ERC721: transfer to non ERC721Receiver implementer");
    }

    /**
     * @dev Returns whether the specified token exists.
     * @param tokenId uint256 ID of the token to query the existence of
     * @return bool whether the token exists
     */
    function _exists(uint256 tokenId) internal view returns (bool) {
        address owner = _tokenOwner[tokenId];
        return owner != address(0);
    }

    /**
     * @dev Returns whether the given spender can transfer a given token ID.
     * @param spender address of the spender to query
     * @param tokenId uint256 ID of the token to be transferred
     * @return bool whether the msg.sender is approved for the given token ID,
     * is an operator of the owner, or is the owner of the token
     */
    function _isApprovedOrOwner(address spender, uint256 tokenId) internal view returns (bool) {
        require(_exists(tokenId), "ERC721: operator query for nonexistent token");
        address owner = ownerOf(tokenId);
        return (spender == owner || getApproved(tokenId) == spender || isApprovedForAll(owner, spender));
    }

    /**
     * @dev Internal function to mint a new token.
     * Reverts if the given token ID already exists.
     * @param to The address that will own the minted token
     * @param tokenId uint256 ID of the token to be minted
     */
    function _mint(address to, uint256 tokenId) internal {
        require(to != address(0), "ERC721: mint to the zero address");
        require(!_exists(tokenId), "ERC721: token already minted");

        _tokenOwner[tokenId] = to;
        _ownedTokensCount[to].increment();

        emit Transfer(address(0), to, tokenId);
    }

    /**
     * @dev Internal function to burn a specific token.
     * Reverts if the token does not exist.
     * Deprecated, use _burn(uint256) instead.
     * @param owner owner of the token to burn
     * @param tokenId uint256 ID of the token being burned
     */
    function _burn(address owner, uint256 tokenId) internal {
        require(ownerOf(tokenId) == owner, "ERC721: burn of token that is not own");

        _clearApproval(tokenId);

        _ownedTokensCount[owner].decrement();
        _tokenOwner[tokenId] = address(0);

        emit Transfer(owner, address(0), tokenId);
    }

    /**
     * @dev Internal function to burn a specific token.
     * Reverts if the token does not exist.
     * @param tokenId uint256 ID of the token being burned
     */
    function _burn(uint256 tokenId) internal {
        _burn(ownerOf(tokenId), tokenId);
    }

    /**
     * @dev Internal function to transfer ownership of a given token ID to another address.
     * As opposed to transferFrom, this imposes no restrictions on msg.sender.
     * @param from current owner of the token
     * @param to address to receive the ownership of the given token ID
     * @param tokenId uint256 ID of the token to be transferred
     */
    function _transferFrom(address from, address to, uint256 tokenId) internal {
        require(ownerOf(tokenId) == from, "ERC721: transfer of token that is not own");
        require(to != address(0), "ERC721: transfer to the zero address");

        _clearApproval(tokenId);

        _ownedTokensCount[from].decrement();
        _ownedTokensCount[to].increment();

        _tokenOwner[tokenId] = to;

        emit Transfer(from, to, tokenId);
    }

    /**
     * @dev Internal function to invoke `onERC721Received` on a target address.
     * The call is not executed if the target address is not a contract.
     *
     * This function is deprecated.
     * @param from address representing the previous owner of the given token ID
     * @param to target address that will receive the tokens
     * @param tokenId uint256 ID of the token to be transferred
     * @param _data bytes optional data to send along with the call
     * @return bool whether the call correctly returned the expected magic value
     */
    function _checkOnERC721Received(address from, address to, uint256 tokenId, bytes memory _data)
        internal returns (bool)
    {
        if (!to.isContract()) {
            return true;
        }

        bytes4 retval = IERC721Receiver(to).onERC721Received(msg.sender, from, tokenId, _data);
        return (retval == _ERC721_RECEIVED);
    }

    /**
     * @dev Private function to clear current approval of a given token ID.
     * @param tokenId uint256 ID of the token to be transferred
     */
    function _clearApproval(uint256 tokenId) private {
        if (_tokenApprovals[tokenId] != address(0)) {
            _tokenApprovals[tokenId] = address(0);
        }
    }
}


/**
 * @title ERC-721 Non-Fungible Token Standard, full implementation interface
 * @dev See https://eips.ethereum.org/EIPS/eip-721
 */
contract IERC721Full is IERC721, IERC721Enumerable, IERC721Metadata {
    // solhint-disable-previous-line no-empty-blocks
}


/**
 * @title ERC-721 Non-Fungible Token with optional enumeration extension logic
 * @dev See https://eips.ethereum.org/EIPS/eip-721
 */
contract ERC721Enumerable is ERC165, ERC721, IERC721Enumerable {
    // Mapping from owner to list of owned token IDs
    mapping(address => uint256[]) private _ownedTokens;

    // Mapping from token ID to index of the owner tokens list
    mapping(uint256 => uint256) private _ownedTokensIndex;

    // Array with all token ids, used for enumeration
    uint256[] private _allTokens;

    // Mapping from token id to position in the allTokens array
    mapping(uint256 => uint256) private _allTokensIndex;

    /*
     *     bytes4(keccak256('totalSupply()')) == 0x18160ddd
     *     bytes4(keccak256('tokenOfOwnerByIndex(address,uint256)')) == 0x2f745c59
     *     bytes4(keccak256('tokenByIndex(uint256)')) == 0x4f6ccce7
     *
     *     => 0x18160ddd ^ 0x2f745c59 ^ 0x4f6ccce7 == 0x780e9d63
     */
    bytes4 private constant _INTERFACE_ID_ERC721_ENUMERABLE = 0x780e9d63;

    /**
     * @dev Constructor function.
     */
    constructor () public {
        // register the supported interface to conform to ERC721Enumerable via ERC165
        _registerInterface(_INTERFACE_ID_ERC721_ENUMERABLE);
    }

    /**
     * @dev Gets the token ID at a given index of the tokens list of the requested owner.
     * @param owner address owning the tokens list to be accessed
     * @param index uint256 representing the index to be accessed of the requested tokens list
     * @return uint256 token ID at the given index of the tokens list owned by the requested address
     */
    function tokenOfOwnerByIndex(address owner, uint256 index) public view returns (uint256) {
        require(index < balanceOf(owner), "ERC721Enumerable: owner index out of bounds");
        return _ownedTokens[owner][index];
    }

    /**
     * @dev Gets the total amount of tokens stored by the contract.
     * @return uint256 representing the total amount of tokens
     */
    function totalSupply() public view returns (uint256) {
        return _allTokens.length;
    }

    /**
     * @dev Gets the token ID at a given index of all the tokens in this contract
     * Reverts if the index is greater or equal to the total number of tokens.
     * @param index uint256 representing the index to be accessed of the tokens list
     * @return uint256 token ID at the given index of the tokens list
     */
    function tokenByIndex(uint256 index) public view returns (uint256) {
        require(index < totalSupply(), "ERC721Enumerable: global index out of bounds");
        return _allTokens[index];
    }

    /**
     * @dev Internal function to transfer ownership of a given token ID to another address.
     * As opposed to transferFrom, this imposes no restrictions on msg.sender.
     * @param from current owner of the token
     * @param to address to receive the ownership of the given token ID
     * @param tokenId uint256 ID of the token to be transferred
     */
    function _transferFrom(address from, address to, uint256 tokenId) internal {
        super._transferFrom(from, to, tokenId);

        _removeTokenFromOwnerEnumeration(from, tokenId);

        _addTokenToOwnerEnumeration(to, tokenId);
    }

    /**
     * @dev Internal function to mint a new token.
     * Reverts if the given token ID already exists.
     * @param to address the beneficiary that will own the minted token
     * @param tokenId uint256 ID of the token to be minted
     */
    function _mint(address to, uint256 tokenId) internal {
        super._mint(to, tokenId);

        _addTokenToOwnerEnumeration(to, tokenId);

        _addTokenToAllTokensEnumeration(tokenId);
    }

    /**
     * @dev Internal function to burn a specific token.
     * Reverts if the token does not exist.
     * Deprecated, use _burn(uint256) instead.
     * @param owner owner of the token to burn
     * @param tokenId uint256 ID of the token being burned
     */
    function _burn(address owner, uint256 tokenId) internal {
        super._burn(owner, tokenId);

        _removeTokenFromOwnerEnumeration(owner, tokenId);
        // Since tokenId will be deleted, we can clear its slot in _ownedTokensIndex to trigger a gas refund
        _ownedTokensIndex[tokenId] = 0;

        _removeTokenFromAllTokensEnumeration(tokenId);
    }

    /**
     * @dev Gets the list of token IDs of the requested owner.
     * @param owner address owning the tokens
     * @return uint256[] List of token IDs owned by the requested address
     */
    function _tokensOfOwner(address owner) internal view returns (uint256[] storage) {
        return _ownedTokens[owner];
    }

    /**
     * @dev Private function to add a token to this extension's ownership-tracking data structures.
     * @param to address representing the new owner of the given token ID
     * @param tokenId uint256 ID of the token to be added to the tokens list of the given address
     */
    function _addTokenToOwnerEnumeration(address to, uint256 tokenId) private {
        _ownedTokensIndex[tokenId] = _ownedTokens[to].length;
        _ownedTokens[to].push(tokenId);
    }

    /**
     * @dev Private function to add a token to this extension's token tracking data structures.
     * @param tokenId uint256 ID of the token to be added to the tokens list
     */
    function _addTokenToAllTokensEnumeration(uint256 tokenId) private {
        _allTokensIndex[tokenId] = _allTokens.length;
        _allTokens.push(tokenId);
    }

    /**
     * @dev Private function to remove a token from this extension's ownership-tracking data structures. Note that
     * while the token is not assigned a new owner, the _ownedTokensIndex mapping is _not_ updated: this allows for
     * gas optimizations e.g. when performing a transfer operation (avoiding double writes).
     * This has O(1) time complexity, but alters the order of the _ownedTokens array.
     * @param from address representing the previous owner of the given token ID
     * @param tokenId uint256 ID of the token to be removed from the tokens list of the given address
     */
    function _removeTokenFromOwnerEnumeration(address from, uint256 tokenId) private {
        // To prevent a gap in from's tokens array, we store the last token in the index of the token to delete, and
        // then delete the last slot (swap and pop).

        uint256 lastTokenIndex = _ownedTokens[from].length.sub(1);
        uint256 tokenIndex = _ownedTokensIndex[tokenId];

        // When the token to delete is the last token, the swap operation is unnecessary
        if (tokenIndex != lastTokenIndex) {
            uint256 lastTokenId = _ownedTokens[from][lastTokenIndex];

            _ownedTokens[from][tokenIndex] = lastTokenId; // Move the last token to the slot of the to-delete token
            _ownedTokensIndex[lastTokenId] = tokenIndex; // Update the moved token's index
        }

        // This also deletes the contents at the last position of the array
        _ownedTokens[from].length--;

        // Note that _ownedTokensIndex[tokenId] hasn't been cleared: it still points to the old slot (now occupied by
        // lastTokenId, or just over the end of the array if the token was the last one).
    }

    /**
     * @dev Private function to remove a token from this extension's token tracking data structures.
     * This has O(1) time complexity, but alters the order of the _allTokens array.
     * @param tokenId uint256 ID of the token to be removed from the tokens list
     */
    function _removeTokenFromAllTokensEnumeration(uint256 tokenId) private {
        // To prevent a gap in the tokens array, we store the last token in the index of the token to delete, and
        // then delete the last slot (swap and pop).

        uint256 lastTokenIndex = _allTokens.length.sub(1);
        uint256 tokenIndex = _allTokensIndex[tokenId];

        // When the token to delete is the last token, the swap operation is unnecessary. However, since this occurs so
        // rarely (when the last minted token is burnt) that we still do the swap here to avoid the gas cost of adding
        // an 'if' statement (like in _removeTokenFromOwnerEnumeration)
        uint256 lastTokenId = _allTokens[lastTokenIndex];

        _allTokens[tokenIndex] = lastTokenId; // Move the last token to the slot of the to-delete token
        _allTokensIndex[lastTokenId] = tokenIndex; // Update the moved token's index

        // This also deletes the contents at the last position of the array
        _allTokens.length--;
        _allTokensIndex[tokenId] = 0;
    }
}


contract ERC721Metadata is ERC165, ERC721, IERC721Metadata {
    // Token name
    string private _name;

    // Token symbol
    string private _symbol;

    // Optional mapping for token URIs
    mapping(uint256 => string) private _tokenURIs;

    /*
     *     bytes4(keccak256('name()')) == 0x06fdde03
     *     bytes4(keccak256('symbol()')) == 0x95d89b41
     *     bytes4(keccak256('tokenURI(uint256)')) == 0xc87b56dd
     *
     *     => 0x06fdde03 ^ 0x95d89b41 ^ 0xc87b56dd == 0x5b5e139f
     */
    bytes4 private constant _INTERFACE_ID_ERC721_METADATA = 0x5b5e139f;

    /**
     * @dev Constructor function
     */
    constructor (string memory name, string memory symbol) public {
        _name = name;
        _symbol = symbol;

        // register the supported interfaces to conform to ERC721 via ERC165
        _registerInterface(_INTERFACE_ID_ERC721_METADATA);
    }

    /**
     * @dev Gets the token name.
     * @return string representing the token name
     */
    function name() external view returns (string memory) {
        return _name;
    }

    /**
     * @dev Gets the token symbol.
     * @return string representing the token symbol
     */
    function symbol() external view returns (string memory) {
        return _symbol;
    }

    /**
     * @dev Returns an URI for a given token ID.
     * Throws if the token ID does not exist. May return an empty string.
     * @param tokenId uint256 ID of the token to query
     */
    function tokenURI(uint256 tokenId) external view returns (string memory) {
        require(_exists(tokenId), "ERC721Metadata: URI query for nonexistent token");
        return _tokenURIs[tokenId];
    }

    /**
     * @dev Internal function to set the token URI for a given token.
     * Reverts if the token ID does not exist.
     * @param tokenId uint256 ID of the token to set its URI
     * @param uri string URI to assign
     */
    function _setTokenURI(uint256 tokenId, string memory uri) internal {
        require(_exists(tokenId), "ERC721Metadata: URI set of nonexistent token");
        _tokenURIs[tokenId] = uri;
    }

    /**
     * @dev Internal function to burn a specific token.
     * Reverts if the token does not exist.
     * Deprecated, use _burn(uint256) instead.
     * @param owner owner of the token to burn
     * @param tokenId uint256 ID of the token being burned by the msg.sender
     */
    function _burn(address owner, uint256 tokenId) internal {
        super._burn(owner, tokenId);

        // Clear metadata (if any)
        if (bytes(_tokenURIs[tokenId]).length != 0) {
            delete _tokenURIs[tokenId];
        }
    }
}


/**
 * @title Full ERC721 Token
 * This implementation includes all the required and some optional functionality of the ERC721 standard
 * Moreover, it includes approve all functionality using operator terminology
 * @dev see https://eips.ethereum.org/EIPS/eip-721
 */
contract ERC721Full is ERC721, ERC721Enumerable, ERC721Metadata {
    constructor (string memory name, string memory symbol) public ERC721Metadata(name, symbol) {
        // solhint-disable-previous-line no-empty-blocks
    }
}


/**
 * @author Ricardo Guilherme Schmidt (Status Research & Development GmbH)
 */
contract StickerPack is Controlled, TokenClaimer, ERC721Full("Status Sticker Pack","STKP") {

    mapping(uint256 => uint256) public tokenPackId; //packId
    uint256 public tokenCount; //tokens buys

    /**
     * @notice controller can generate tokens at will
     * @param _owner account being included new token
     * @param _packId pack being minted
     * @return tokenId created
     */
    function generateToken(address _owner, uint256 _packId)
        external
        onlyController
        returns (uint256 tokenId)
    {
        tokenId = tokenCount++;
        tokenPackId[tokenId] = _packId;
        _mint(_owner, tokenId);
    }

    /**
     * @notice This method can be used by the controller to extract mistakenly
     *  sent tokens to this contract.
     * @param _token The address of the token contract that you want to recover
     *  set to 0 in case you want to extract ether.
     */
    function claimTokens(address _token)
        external
        onlyController
    {
        withdrawBalance(_token, controller);
    }



}


interface ApproveAndCallFallBack {
    function receiveApproval(address from, uint256 _amount, address _token, bytes calldata _data) external;
}




/**
 * @author Ricardo Guilherme Schmidt (Status Research & Development GmbH) 
 * StickerMarket allows any address register "StickerPack" which can be sold to any address in form of "StickerPack", an ERC721 token.
 */
contract StickerMarket is Controlled, TokenClaimer, ApproveAndCallFallBack {
    using SafeMath for uint256;
    
    event ClaimedTokens(address indexed _token, address indexed _controller, uint256 _amount);
    event MarketState(State state);
    event RegisterFee(uint256 value);
    event BurnRate(uint256 value);

    enum State { Invalid, Open, BuyOnly, Controlled, Closed }

    State public state = State.Open;
    uint256 registerFee;
    uint256 burnRate;
    
    //include global var to set burn rate/percentage
    ERC20Token public snt; //payment token
    StickerPack public stickerPack;
    StickerType public stickerType;
    
    /**
     * @dev can only be called when market is open or by controller on Controlled state
     */
    modifier marketManagement {
        require(state == State.Open || (msg.sender == controller && state == State.Controlled), "Market Disabled");
        _;
    }

    /**
     * @dev can only be called when market is open or buy-only state.
     */
    modifier marketSell {
        require(state == State.Open || state == State.BuyOnly || (msg.sender == controller && state == State.Controlled), "Market Disabled");
        _;
    }

    /**
     * @param _snt SNT token
     */
    constructor(
        ERC20Token _snt,
        StickerPack _stickerPack,
        StickerType _stickerType
    ) 
        public
    { 
        require(address(_snt) != address(0), "Bad _snt parameter");
        require(address(_stickerPack) != address(0), "Bad _stickerPack parameter");
        require(address(_stickerType) != address(0), "Bad _stickerType parameter");
        snt = _snt;
        stickerPack = _stickerPack;
        stickerType = _stickerType;
    }

    /** 
     * @dev Mints NFT StickerPack in `msg.sender` account, and Transfers SNT using user allowance
     * emit NonfungibleToken.Transfer(`address(0)`, `msg.sender`, `tokenId`)
     * @notice buy a pack from market pack owner, including a StickerPack's token in msg.sender account with same metadata of `_packId` 
     * @param _packId id of market pack 
     * @param _destination owner of token being brought
     * @param _price agreed price 
     * @return tokenId generated StickerPack token 
     */
    function buyToken(
        uint256 _packId,
        address _destination,
        uint256 _price
    ) 
        external  
        returns (uint256 tokenId)
    {
        return buy(msg.sender, _packId, _destination, _price);
    }

    /** 
     * @dev emits StickerMarket.Register(`packId`, `_urlHash`, `_price`, `_contenthash`)
     * @notice Registers to sell a sticker pack 
     * @param _price cost in wei to users minting this pack
     * @param _donate value between 0-10000 representing percentage of `_price` that is donated to StickerMarket at every buy
     * @param _category listing category
     * @param _owner address of the beneficiary of buys
     * @param _contenthash EIP1577 pack contenthash for listings
     * @param _fee Fee msg.sender agrees to pay for this registration
     * @return packId Market position of Sticker Pack data.
     */
    function registerPack(
        uint256 _price,
        uint256 _donate,
        bytes4[] calldata _category, 
        address _owner,
        bytes calldata _contenthash,
        uint256 _fee
    ) 
        external  
        returns(uint256 packId)
    {
        packId = register(msg.sender, _category, _owner, _price, _donate, _contenthash, _fee);
    }

    /**
     * @notice MiniMeToken ApproveAndCallFallBack forwarder for registerPack and buyToken
     * @param _from account calling "approve and buy" 
     * @param _value must be exactly whats being consumed     
     * @param _token must be exactly SNT contract
     * @param _data abi encoded call 
     */
    function receiveApproval(
        address _from,
        uint256 _value,
        address _token,
        bytes calldata _data
    ) 
        external 
    {
        require(_token == address(snt), "Bad token");
        require(_token == address(msg.sender), "Bad call");
        bytes4 sig = abiDecodeSig(_data);
        bytes memory cdata = slice(_data,4,_data.length-4);
        if(sig == this.buyToken.selector){
            require(cdata.length == 96, "Bad data length");
            (uint256 packId, address owner, uint256 price) = abi.decode(cdata, (uint256, address, uint256));
            require(_value == price, "Bad price value");
            buy(_from, packId, owner, price);
        } else if(sig == this.registerPack.selector) {
            require(cdata.length >= 188, "Bad data length");
            (uint256 price, uint256 donate, bytes4[] memory category, address owner, bytes memory contenthash, uint256 fee) = abi.decode(cdata, (uint256,uint256,bytes4[],address,bytes,uint256));
            require(_value == fee, "Bad fee value");
            register(_from, category, owner, price, donate, contenthash, fee);
        } else {
            revert("Bad call");
        }
    }

    /**
     * @notice changes market state, only controller can call.
     * @param _state new state
     */
    function setMarketState(State _state)
        external
        onlyController 
    {
        state = _state;
        emit MarketState(_state);
    }

    /**
     * @notice changes register fee, only controller can call.
     * @param _value total SNT cost of registration
     */
    function setRegisterFee(uint256 _value)
        external
        onlyController 
    {
        registerFee = _value;
        emit RegisterFee(_value);
    }

    /**
     * @notice changes burn rate percentage, only controller can call.
     * @param _value new value between 0 and 10000
     */
    function setBurnRate(uint256 _value)
        external
        onlyController 
    {
        burnRate = _value;
        require(_value <= 10000, "cannot be more then 100.00%");
        emit BurnRate(_value);
    }
    
    /** 
     * @notice controller can generate packs at will
     * @param _price cost in wei to users minting with _urlHash metadata
     * @param _donate optional amount of `_price` that is donated to StickerMarket at every buy
     * @param _category listing category
     * @param _owner address of the beneficiary of buys
     * @param _contenthash EIP1577 pack contenthash for listings
     * @return packId Market position of Sticker Pack data.
     */
    function generatePack(
        uint256 _price,
        uint256 _donate,
        bytes4[] calldata _category, 
        address _owner,
        bytes calldata _contenthash
    ) 
        external  
        onlyController
        returns(uint256 packId)
    {
        packId = stickerType.generatePack(_price, _donate, _category, _owner, _contenthash);
    }

    /**
     * @notice removes all market data about a marketed pack, can only be called by market controller
     * @param _packId pack being purged
     * @param _limit limits categories being purged
     */
    function purgePack(uint256 _packId, uint256 _limit)
        external
        onlyController 
    {
        stickerType.purgePack(_packId, _limit);
    }

    /**
     * @notice controller can generate tokens at will
     * @param _owner account being included new token
     * @param _packId pack being minted
     * @return tokenId created
     */
    function generateToken(address _owner, uint256 _packId) 
        external
        onlyController 
        returns (uint256 tokenId)
    {
        return stickerPack.generateToken(_owner, _packId);
    }

    /**
     * @notice Change controller of stickerType
     * @param _newController new controller of stickerType.
     */
    function migrate(address payable _newController) 
        external
        onlyController 
    {
        require(_newController != address(0), "Cannot unset controller");
        stickerType.changeController(_newController);
        stickerPack.changeController(_newController);
    }

    /**
     * @notice This method can be used by the controller to extract mistakenly
     *  sent tokens to this contract.
     * @param _token The address of the token contract that you want to recover
     *  set to 0 in case you want to extract ether.
     */
    function claimTokens(address _token) 
        external
        onlyController 
    {
        withdrawBalance(_token, controller);
    }

    /**
     * @notice returns pack data of token
     * @param _tokenId user token being queried
     * @return categories, registration time and contenthash
     */
    function getTokenData(uint256 _tokenId) 
        external 
        view 
        returns (
            bytes4[] memory category,
            uint256 timestamp,
            bytes memory contenthash
        ) 
    {
        return stickerType.getPackSummary(stickerPack.tokenPackId(_tokenId));
    }

    /** 
     * @dev charges registerFee and register new pack to owner
     * @param _caller payment account
     * @param _category listing category
     * @param _owner address of the beneficiary of buys
     * @param _price cost in wei to users minting this pack
     * @param _donate value between 0-10000 representing percentage of `_price` that is donated to StickerMarket at every buy
     * @param _contenthash EIP1577 pack contenthash for listings
     * @param _fee Fee msg.sender agrees to pay for this registrion
     * @return created packId
     */
    function register(
        address _caller,
        bytes4[] memory _category,
        address _owner,
        uint256 _price,
        uint256 _donate,
        bytes memory _contenthash,
        uint256 _fee
    ) 
        internal 
        marketManagement
        returns(uint256 packId) 
    {
        require(_fee == registerFee, "Unexpected fee");
        if(registerFee > 0){
            require(snt.transferFrom(_caller, controller, registerFee), "Bad payment");
        }
        packId = stickerType.generatePack(_price, _donate, _category, _owner, _contenthash);
    }

    /** 
     * @dev transfer SNT from buyer to pack owner and mint sticker pack token 
     * @param _caller payment account
     * @param _packId id of market pack 
     * @param _destination owner of token being brought
     * @param _price agreed price 
     * @return created tokenId
     */
    function buy(
        address _caller,
        uint256 _packId,
        address _destination,
        uint256 _price
    ) 
        internal 
        marketSell
        returns (uint256 tokenId)
    {
        (
            address pack_owner,
            bool pack_mintable,
            uint256 pack_price,
            uint256 pack_donate
        ) = stickerType.getPaymentData(_packId);
        require(pack_owner != address(0), "Bad pack");
        require(pack_mintable, "Disabled");
        uint256 amount = pack_price;
        require(_price == amount, "Wrong price");
        require(amount > 0, "Unauthorized");
        if(amount > 0 && burnRate > 0) {
            uint256 burned = amount.mul(burnRate).div(10000);
            amount = amount.sub(burned);
            require(snt.transferFrom(_caller, Controlled(address(snt)).controller(), burned), "Bad burn");
        }
        if(amount > 0 && pack_donate > 0) {
            uint256 donate = amount.mul(pack_donate).div(10000);
            amount = amount.sub(donate);
            require(snt.transferFrom(_caller, controller, donate), "Bad donate");
        } 
        if(amount > 0) {
            require(snt.transferFrom(_caller, pack_owner, amount), "Bad payment");
        }
        return stickerPack.generateToken(_destination, _packId);
    }

    /**
     * @dev decodes sig of abi encoded call
     * @param _data abi encoded data
     * @return sig (first 4 bytes)
     */
    function abiDecodeSig(bytes memory _data) private pure returns(bytes4 sig){
        assembly {
            sig := mload(add(_data, add(0x20, 0)))
        }
    }

    /**
     * @dev get a slice of byte array
     * @param _bytes source
     * @param _start pointer
     * @param _length size to read
     * @return sliced bytes
     */
    function slice(bytes memory _bytes, uint _start, uint _length) private pure returns (bytes memory) {
        require(_bytes.length >= (_start + _length));

        bytes memory tempBytes;

        assembly {
            switch iszero(_length)
            case 0 {
                // Get a location of some free memory and store it in tempBytes as
                // Solidity does for memory variables.
                tempBytes := mload(0x40)

                // The first word of the slice result is potentially a partial
                // word read from the original array. To read it, we calculate
                // the length of that partial word and start copying that many
                // bytes into the array. The first word we copy will start with
                // data we don't care about, but the last `lengthmod` bytes will
                // land at the beginning of the contents of the new array. When
                // we're done copying, we overwrite the full first word with
                // the actual length of the slice.
                let lengthmod := and(_length, 31)

                // The multiplication in the next line is necessary
                // because when slicing multiples of 32 bytes (lengthmod == 0)
                // the following copy loop was copying the origin's length
                // and then ending prematurely not copying everything it should.
                let mc := add(add(tempBytes, lengthmod), mul(0x20, iszero(lengthmod)))
                let end := add(mc, _length)

                for {
                    // The multiplication in the next line has the same exact purpose
                    // as the one above.
                    let cc := add(add(add(_bytes, lengthmod), mul(0x20, iszero(lengthmod))), _start)
                } lt(mc, end) {
                    mc := add(mc, 0x20)
                    cc := add(cc, 0x20)
                } {
                    mstore(mc, mload(cc))
                }

                mstore(tempBytes, _length)

                //update free-memory pointer
                //allocating the array padded to 32 bytes like the compiler does now
                mstore(0x40, and(add(mc, 31), not(31)))
            }
            //if we want a zero-length slice let's just return a zero-length array
            default {
                tempBytes := mload(0x40)

                mstore(0x40, add(tempBytes, 0x20))
            }
        }

        return tempBytes;
    }


    // For ABI/web3.js purposes
    // fired by StickerType
    event Register(uint256 indexed packId, uint256 dataPrice, bytes contenthash);
    // fired by StickerPack and MiniMeToken
      event Transfer(
        address indexed from,
        address indexed to,
        uint256 indexed value
    );
}



/**
 * @author Ricardo Guilherme Schmidt (Status Research & Development GmbH)
 * StickerMarket allows any address register "StickerPack" which can be sold to any address in form of "StickerPack", an ERC721 token.
 */
contract StickerType is Controlled, TokenClaimer, ERC721Full("Status Sticker Pack Authorship","STKA") {
    using SafeMath for uint256;
    event Register(uint256 indexed packId, uint256 dataPrice, bytes contenthash, bool mintable);
    event PriceChanged(uint256 indexed packId, uint256 dataPrice);
    event MintabilityChanged(uint256 indexed packId, bool mintable);
    event ContenthashChanged(uint256 indexed packid, bytes contenthash);
    event Categorized(bytes4 indexed category, uint256 indexed packId);
    event Uncategorized(bytes4 indexed category, uint256 indexed packId);
    event Unregister(uint256 indexed packId);

    struct Pack {
        bytes4[] category;
        bool mintable;
        uint256 timestamp;
        uint256 price; //in "wei"
        uint256 donate; //in "percent"
        bytes contenthash;
    }

    uint256 registerFee;
    uint256 burnRate;

    mapping(uint256 => Pack) public packs;
    uint256 public packCount; //pack registers


    //auxilary views
    mapping(bytes4 => uint256[]) private availablePacks; //array of available packs
    mapping(bytes4 => mapping(uint256 => uint256)) private availablePacksIndex; //position on array of available packs
    mapping(uint256 => mapping(bytes4 => uint256)) private packCategoryIndex;

    /**
     * Can only be called by the pack owner, or by the controller if pack exists.
     */
    modifier packOwner(uint256 _packId) {
        address owner = ownerOf(_packId);
        require((msg.sender == owner) || (owner != address(0) && msg.sender == controller), "Unauthorized");
        _;
    }

    /**
     * @notice controller can generate packs at will
     * @param _price cost in wei to users minting with _urlHash metadata
     * @param _donate optional amount of `_price` that is donated to StickerMarket at every buy
     * @param _category listing category
     * @param _owner address of the beneficiary of buys
     * @param _contenthash EIP1577 pack contenthash for listings
     * @return packId Market position of Sticker Pack data.
     */
    function generatePack(
        uint256 _price,
        uint256 _donate,
        bytes4[] calldata _category,
        address _owner,
        bytes calldata _contenthash
    )
        external
        onlyController
        returns(uint256 packId)
    {
        require(_donate <= 10000, "Bad argument, _donate cannot be more then 100.00%");
        packId = packCount++;
        _mint(_owner, packId);
        packs[packId] = Pack(new bytes4[](0), true, block.timestamp, _price, _donate, _contenthash);
        emit Register(packId, _price, _contenthash, true);
        for(uint i = 0;i < _category.length; i++){
            addAvailablePack(packId, _category[i]);
        }
    }

    /**
     * @notice removes all market data about a marketed pack, can only be called by market controller
     * @param _packId position to be deleted
     * @param _limit limit of categories to cleanup
     */
    function purgePack(uint256 _packId, uint256 _limit)
        external
        onlyController
    {
        bytes4[] memory _category = packs[_packId].category;
        uint limit;
        if(_limit == 0) {
            limit = _category.length;
        } else {
            require(_limit <= _category.length, "Bad limit");
            limit = _limit;
        }

        uint256 len = _category.length;
        if(len > 0){
            len--;
        }
        for(uint i = 0; i < limit; i++){
            removeAvailablePack(_packId, _category[len-i]);
        }

        if(packs[_packId].category.length == 0){
            _burn(ownerOf(_packId), _packId);
            delete packs[_packId];
            emit Unregister(_packId);
        }

    }

    /**
     * @notice changes contenthash of `_packId`, can only be called by controller
     * @param _packId which market position is being altered
     * @param _contenthash new contenthash
     */
    function setPackContenthash(uint256 _packId, bytes calldata _contenthash)
        external
        onlyController
    {
        emit ContenthashChanged(_packId, _contenthash);
        packs[_packId].contenthash = _contenthash;
    }

    /**
     * @notice This method can be used by the controller to extract mistakenly
     *  sent tokens to this contract.
     * @param _token The address of the token contract that you want to recover
     *  set to 0 in case you want to extract ether.
     */
    function claimTokens(address _token)
        external
        onlyController
    {
        withdrawBalance(_token, controller);
    }

    /**
     * @notice changes price of `_packId`, can only be called when market is open
     * @param _packId pack id changing price settings
     * @param _price cost in wei to users minting this pack
     * @param _donate value between 0-10000 representing percentage of `_price` that is donated to StickerMarket at every buy
     */
    function setPackPrice(uint256 _packId, uint256 _price, uint256 _donate)
        external
        packOwner(_packId)
    {
        require(_donate <= 10000, "Bad argument, _donate cannot be more then 100.00%");
        emit PriceChanged(_packId, _price);
        packs[_packId].price = _price;
        packs[_packId].donate = _donate;
    }

    /**
     * @notice add caregory in `_packId`, can only be called when market is open
     * @param _packId pack adding category
     * @param _category category to list
     */
    function addPackCategory(uint256 _packId, bytes4 _category)
        external
        packOwner(_packId)
    {
        addAvailablePack(_packId, _category);
    }

    /**
     * @notice remove caregory in `_packId`, can only be called when market is open
     * @param _packId pack removing category
     * @param _category category to unlist
     */
    function removePackCategory(uint256 _packId, bytes4 _category)
        external
        packOwner(_packId)
    {
        removeAvailablePack(_packId, _category);
    }

    /**
     * @notice Changes if pack is enabled for sell
     * @param _packId position edit
     * @param _mintable true to enable sell
     */
    function setPackState(uint256 _packId, bool _mintable)
        external
        packOwner(_packId)
    {
        emit MintabilityChanged(_packId, _mintable);
        packs[_packId].mintable = _mintable;
    }

    /**
     * @notice read available market ids in a category (might be slow)
     * @param _category listing category
     * @return array of market id registered
     */
    function getAvailablePacks(bytes4 _category)
        external
        view
        returns (uint256[] memory availableIds)
    {
        return availablePacks[_category];
    }

    /**
     * @notice count total packs in a category
     * @param _category listing category
     * @return total number of packs in category
     */
    function getCategoryLength(bytes4 _category)
        external
        view
        returns (uint256 size)
    {
        size = availablePacks[_category].length;
    }

    /**
     * @notice read a packId in the category list at a specific index
     * @param _category listing category
     * @param _index index
     * @return packId on index
     */
    function getCategoryPack(bytes4 _category, uint256 _index)
        external
        view
        returns (uint256 packId)
    {
        packId = availablePacks[_category][_index];
    }

    /**
     * @notice returns all data from pack in market
     * @param _packId pack id being queried
     * @return categories, owner, mintable, price, donate and contenthash
     */
    function getPackData(uint256 _packId)
        external
        view
        returns (
            bytes4[] memory category,
            address owner,
            bool mintable,
            uint256 timestamp,
            uint256 price,
            bytes memory contenthash
        )
    {
        Pack memory pack = packs[_packId];
        return (
            pack.category,
            ownerOf(_packId),
            pack.mintable,
            pack.timestamp,
            pack.price,
            pack.contenthash
        );
    }

    /**
     * @notice returns all data from pack in market
     * @param _packId pack id being queried
     * @return categories, owner, mintable, price, donate and contenthash
     */
    function getPackSummary(uint256 _packId)
        external
        view
        returns (
            bytes4[] memory category,
            uint256 timestamp,
            bytes memory contenthash
        )
    {
        Pack memory pack = packs[_packId];
        return (
            pack.category,
            pack.timestamp,
            pack.contenthash
        );
    }

    /**
     * @notice returns payment data for migrated contract
     * @param _packId pack id being queried
     * @return owner, mintable, price and donate
     */
    function getPaymentData(uint256 _packId)
        external
        view
        returns (
            address owner,
            bool mintable,
            uint256 price,
            uint256 donate
        )
    {
        Pack memory pack = packs[_packId];
        return (
            ownerOf(_packId),
            pack.mintable,
            pack.price,
            pack.donate
        );
    }

    /**
     * @dev adds id from "available list"
     * @param _packId altered pack
     * @param _category listing category
     */
    function addAvailablePack(uint256 _packId, bytes4 _category) private {
        require(packCategoryIndex[_packId][_category] == 0, "Duplicate categorization");
        availablePacksIndex[_category][_packId] = availablePacks[_category].push(_packId);
        packCategoryIndex[_packId][_category] = packs[_packId].category.push(_category);
        emit Categorized(_category, _packId);
    }

    /**
     * @dev remove id from "available list"
     * @param _packId altered pack
     * @param _category listing category
     */
    function removeAvailablePack(uint256 _packId, bytes4 _category) private {
        uint pos = availablePacksIndex[_category][_packId];
        require(pos > 0, "Not categorized [1]");
        delete availablePacksIndex[_category][_packId];
        if(pos != availablePacks[_category].length){
            uint256 movedElement = availablePacks[_category][availablePacks[_category].length-1]; //tokenId;
            availablePacks[_category][pos-1] = movedElement;
            availablePacksIndex[_category][movedElement] = pos;
        }
        availablePacks[_category].length--;

        uint pos2 = packCategoryIndex[_packId][_category];
        require(pos2 > 0, "Not categorized [2]");
        delete packCategoryIndex[_packId][_category];
        if(pos2 != packs[_packId].category.length){
            bytes4 movedElement2 = packs[_packId].category[packs[_packId].category.length-1]; //tokenId;
            packs[_packId].category[pos2-1] = movedElement2;
            packCategoryIndex[_packId][movedElement2] = pos2;
        }
        packs[_packId].category.length--;
        emit Uncategorized(_category, _packId);

    }

}