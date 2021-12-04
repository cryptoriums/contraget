/**
 *Submitted for verification at Etherscan.io on 2021-09-15
 */

pragma solidity 0.8.10;

contract SimpleStorage {
    string ipfsHash;
    event StorageSet(string);

    function set(string memory x) public {
        ipfsHash = x;
        emit StorageSet(x);
    }

    function get() public view returns (string memory) {
        return ipfsHash;
    }
}
