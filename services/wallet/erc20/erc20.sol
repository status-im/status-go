pragma solidity ^0.5.0;

contract ERC20Transfer {

  constructor() public {}

  event Transfer(address indexed from, address indexed to, uint256 value);

  function transfer(address to, uint256 value) public {
    emit Transfer(msg.sender, to, value);
  }
}
