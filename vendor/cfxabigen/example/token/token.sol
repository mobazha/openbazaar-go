// SPDX-License-Identifier: MIT
pragma solidity >=0.4.22 <0.9.0;

contract MyToken {
    /* Public variables of the token */
    string public name;
    string public symbol;
    uint8 public decimals;

    /* This creates an array with all balances */
    mapping(address => uint256) public balanceOf;

    /* This generates a public event on the blockchain that will notify clients */
    event Transfer(address indexed from, address indexed to, uint256 value);

    /* Initializes contract with initial supply tokens to the creator of the contract */
    constructor(
        uint256 initialSupply,
        string memory tokenName,
        uint8 decimalUnits,
        string memory tokenSymbol
    ) public {
        balanceOf[msg.sender] = initialSupply; // Give the creator all initial tokens
        name = tokenName; // Set the name for display purposes
        symbol = tokenSymbol; // Set the symbol for display purposes
        decimals = decimalUnits; // Amount of decimals for display purposes
    }

    /* Send coins */
    function transfer(address _to, uint256 _value) public {
        require(balanceOf[msg.sender] > _value, "balance not enough"); // Check if the sender has enough
        require(balanceOf[_to] + _value > balanceOf[_to],"overflow"); // Check for overflows
        balanceOf[msg.sender] -= _value; // Subtract from the sender
        balanceOf[_to] += _value; // Add the same to the recipient
        emit Transfer(msg.sender, _to, _value); // Notify anyone listening that this transfer took place
    }
}
