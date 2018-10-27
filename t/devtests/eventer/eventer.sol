pragma solidity ^0.4.21;

contract Eventer {

    int public crt;

    event Message(
        bytes32 indexed topic,
        int indexed crt
    );

    constructor() public {}

    function emit(bytes32 topic) public {
        crt += 1;
        emit Message(topic, crt);
    }
}
