pragma solidity ^0.7.4;

import { IPoseidonHasher } from "./crypto/PoseidonHasher.sol";

contract RLN {
	uint256 public immutable MEMBERSHIP_DEPOSIT;
	uint256 public immutable DEPTH;
	uint256 public immutable SET_SIZE;

	uint256 public pubkeyIndex = 0;
	mapping(uint256 => uint256) public members;

	IPoseidonHasher public poseidonHasher;

	event MemberRegistered(uint256 pubkey, uint256 index);
	event MemberWithdrawn(uint256 pubkey, uint256 index);

	constructor(
		uint256 membershipDeposit,
		uint256 depth,
		address _poseidonHasher
	) public {
		MEMBERSHIP_DEPOSIT = membershipDeposit;
		DEPTH = depth;
		SET_SIZE = 1 << depth;
		poseidonHasher = IPoseidonHasher(_poseidonHasher);
	}

	function register(uint256 pubkey) external payable {
		require(pubkeyIndex < SET_SIZE, "RLN, register: set is full");
		require(msg.value == MEMBERSHIP_DEPOSIT, "RLN, register: membership deposit is not satisfied");
		_register(pubkey);
	}

	function registerBatch(uint256[] calldata pubkeys) external payable {
		require(pubkeyIndex + pubkeys.length <= SET_SIZE, "RLN, registerBatch: set is full");
		require(msg.value == MEMBERSHIP_DEPOSIT * pubkeys.length, "RLN, registerBatch: membership deposit is not satisfied");
		for (uint256 i = 0; i < pubkeys.length; i++) {
			_register(pubkeys[i]);
		}
	}

	function _register(uint256 pubkey) internal {
		members[pubkeyIndex] = pubkey;
		emit MemberRegistered(pubkey, pubkeyIndex);
		pubkeyIndex += 1;
	}

	function withdrawBatch(
		uint256[] calldata secrets,
		uint256[] calldata pubkeyIndexes,
		address payable[] calldata receivers
	) external {
		uint256 batchSize = secrets.length;
		require(batchSize != 0, "RLN, withdrawBatch: batch size zero");
		require(batchSize == pubkeyIndexes.length, "RLN, withdrawBatch: batch size mismatch pubkey indexes");
		require(batchSize == receivers.length, "RLN, withdrawBatch: batch size mismatch receivers");
		for (uint256 i = 0; i < batchSize; i++) {
			_withdraw(secrets[i], pubkeyIndexes[i], receivers[i]);
		}
	}

	function withdraw(
		uint256 secret,
		uint256 _pubkeyIndex,
		address payable receiver
	) external {
		_withdraw(secret, _pubkeyIndex, receiver);
	}

	function _withdraw(
		uint256 secret,
		uint256 _pubkeyIndex,
		address payable receiver
	) internal {
		require(_pubkeyIndex < SET_SIZE, "RLN, _withdraw: invalid pubkey index");
		require(members[_pubkeyIndex] != 0, "RLN, _withdraw: member doesn't exist");
		require(receiver != address(0), "RLN, _withdraw: empty receiver address");

		// derive public key
		uint256 pubkey = hash([secret, 0]);
		require(members[_pubkeyIndex] == pubkey, "RLN, _withdraw: not verified");

		// delete member
		members[_pubkeyIndex] = 0;

		// refund deposit
		receiver.transfer(MEMBERSHIP_DEPOSIT);

		emit MemberWithdrawn(pubkey, _pubkeyIndex);
	}

	function hash(uint256[2] memory input) internal view returns (uint256) {
		return poseidonHasher.hash(input);
	}
}