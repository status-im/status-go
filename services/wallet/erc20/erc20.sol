pragma solidity ^0.5.0;

contract ERC20Transfer {

  mapping (address => uint256) public balances;

  constructor() public {}

  event Transfer(address indexed from, address indexed to, uint256 value);

  function transfer(address to, uint256 value) public {
    balances[to] += value;
    emit Transfer(msg.sender, to, value);
  }

  function balanceOf(address account) public view returns (uint256) {
    return balances[account];
  }
}
