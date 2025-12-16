// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.20;

contract MockERC721 {
    string public name = "Owner Token";
    string public symbol = "OT";

    mapping(uint256 => address) internal _owners;
    mapping(address => uint256) internal _balances;

    event Transfer(address indexed from, address indexed to, uint256 indexed tokenId);

    constructor() {}

    function ownerOf(uint256 tokenId) public view returns (address owner) {
        owner = _owners[tokenId];
        require(owner != address(0), "ERC721: owner query for nonexistent token");
    }

    function balanceOf(address owner) public view returns (uint256) {
        require(owner != address(0), "ERC721: balance query for the zero address");
        return _balances[owner];
    }

    function mint(address to, uint256 tokenId) public {
        require(to != address(0), "ERC721: mint to the zero address");
        require(_owners[tokenId] == address(0), "ERC721: token already minted");
        _balances[to] += 1;
        _owners[tokenId] = to;
        emit Transfer(address(0), to, tokenId);
    }
}