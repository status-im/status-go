pragma solidity ^0.8.5;

contract Directory {

    bytes[] public communities;

    function isCommunityInDirectory(bytes calldata community) public view returns (bool) { }

    function getCommunities() public view returns (bytes[] memory) { }

    function addCommunity(bytes calldata community) public { }

    function removeCommunity(bytes calldata community) public { }
}
