// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.6.0;

contract TestPrecompiledContract {
    address public constant PACKAGE_VERIFY_CONTRACT = address(0x0000000000000000000000000000000000000066);
    address public constant HEADER_VALIDATE_CONTRACT = address(0x0000000000000000000000000000000000000067);

    function verifyPackage(bytes calldata _payload) public view returns (bool, bytes memory) {
        return PACKAGE_VERIFY_CONTRACT.staticcall(_payload);
    }

    function verifyHeader(bytes calldata _payload) public view returns (bool, bytes memory) {
        return HEADER_VALIDATE_CONTRACT.staticcall(_payload);
    }
}

