require('dotenv').config('../.env');
const {expect} = require('chai');
const {ethers} = require('hardhat');
const {parseEther, formatEther} = require("ethers/lib/utils");
const utils = require('../scripts/utils.js');
const RPC_URL = process.env.RPC_URL;

describe('transfer bep20 Test', function () {
    it('deploy and transfer bep20 token', async function () {
        // init owner and one account
        let account = ["a180523a5ac6cac101155057133c88353f098a05b1bed6f1076f3bc677ed8cd1"]
        let provider = ethers.getDefaultProvider(RPC_URL);
        let accountAddress = new ethers.Wallet(account[0], provider);
        const [owner] = await ethers.getSigners();

        // deploy bep20 contract
        const Token = await ethers.getContractFactory('BEP20Token');
        let tokenYContract = await Token.deploy("Y", "Y token");
        await tokenYContract.deployed();

        // transfer bep20 token from owner
        let value = await tokenYContract.balanceOf(accountAddress.address);
        console.log("balance of account in contractY: ", formatEther(value.toString()));
        let tx = await tokenYContract.connect(owner).transfer(accountAddress.address, parseEther('10000'));
        await utils.checkTransStatus(tx)
        value = await tokenYContract.balanceOf(accountAddress.address)
        expect(value.toString()).equal(parseEther('10000').toString());
        console.log("balance of account in contractY: ", formatEther(value.toString()));
    });
});