const {expect} = require('chai');
var chai = require('chai');
const BN = require('bn.js');
const {ethers} = require('hardhat');
const {formatUnits, formatEther, parseEther, parseUnits} = require("ethers/lib/utils");
chai.use(require('chai-bn')(BN));

describe('swap bep20 Test', function () {
    it('deploy and swap tokens', async function () {
        // deploy UniswapV2Factory
        let account = ["a180523a5ac6cac101155057133c88353f098a05b1bed6f1076f3bc677ed8cd1", "414c619be8210cb18bb209cda801ba56fde57023a7783c9a58a72b7a75d9cc21"]
        const [owner] = await ethers.getSigners();
        let provider = ethers.getDefaultProvider("https://bas-cube-devnet.bk.nodereal.cc");
        let addr1 = new ethers.Wallet(account[0], provider);
        let addr2 = new ethers.Wallet(account[1], provider);
        let accountAddress = [addr1, addr2];
        let Factory = await ethers.getContractFactory('UniswapV2Factory');
        let factoryContract = await Factory.deploy(owner.address);
        await factoryContract.deployed();
        console.log('factory: ' + factoryContract.address);

        let pairInitHash = await factoryContract.getPairInitHash();
        console.log("pairInitHash:", pairInitHash.toString())

        // deploy WBNB
        let WBNB = await ethers.getContractFactory('WBNB');
        let wbnbContract = await WBNB.deploy();
        await wbnbContract.deployed();
        console.log('wbnb: ' + wbnbContract.address);

        // deploy UniswapV2Router02
        let Router = await ethers.getContractFactory('UniswapV2Router02');
        let routerContract = await Router.deploy(factoryContract.address, wbnbContract.address);
        await routerContract.deployed();
        console.log('router: ' + routerContract.address);

        // deploy two BEP20Token contracts
        let bep20TokenContracts = [];
        const Token = await ethers.getContractFactory('BEP20Token');
        let tokenAContract = await Token.deploy("A1", "A1 token");
        await tokenAContract.deployed();
        bep20TokenContracts.push(tokenAContract.address);
        console.log("A" + ": " + tokenAContract.address);
        let tokenBContract = await Token.deploy("B1", "B1 token");
        await tokenBContract.deployed();
        console.log("B" + ": " + tokenBContract.address);
        bep20TokenContracts.push(tokenBContract.address);

        // create pair between two bep20 tokens and wbnb
        let transaction2 = await factoryContract.createPair(bep20TokenContracts[0], wbnbContract.address);
        await checkTransStatus(transaction2);
        console.log('createPair 1 success tx status: 1');

        let transaction3 = await factoryContract.createPair(wbnbContract.address, bep20TokenContracts[1]);
        await checkTransStatus(transaction3);
        console.log('createPair 2 success tx status: 1');

        let transaction = await factoryContract.createPair(bep20TokenContracts[0], bep20TokenContracts[1]);
        await checkTransStatus(transaction);
        console.log('createPair 0  status: 1');
        let pair = await factoryContract.getPair(bep20TokenContracts[0], bep20TokenContracts[1]);
        console.log('index: ' + ' pair: ' + pair);

        const gasPrice = await provider.getGasPrice();
        console.log("gas price: ", gasPrice);

        for (let i = 0; i < accountAddress.length; i++) {
            // send original token
            let tx = await owner.sendTransaction({
                from: owner.address, to: accountAddress[0].address, value: parseEther('1'),
                gasLimit:"45000", gasPrice:gasPrice});
            await checkTransStatus(tx);
            console.log("send original token success")

            // deposit WBNB
            tx = await wbnbContract.connect(owner).deposit({value: "4000000000000000000"});
            await checkTransStatus(tx);
            let value = await wbnbContract.balanceOf(owner.address);
            tx = await wbnbContract.connect(owner).transfer(accountAddress[i].address, parseEther('4'));
            await checkTransStatus(tx);
            value = await wbnbContract.balanceOf(accountAddress[i].address);
            console.log("balance of account in wbnbContract: ", formatEther(value.toString()));
            expect(value.toString()).equal(parseEther('4').toString());

            // deposit tokenA
            tx = await tokenAContract.connect(owner).transfer(accountAddress[i].address, parseEther('10000'));
            await checkTransStatus(tx)
            value = await tokenAContract.balanceOf(accountAddress[i].address);
            console.log("balance of account in contractA: ", formatEther(value.toString()));
            value = await tokenAContract.balanceOf(accountAddress[i].address)
            expect(value.toString()).equal(parseEther('10000').toString());

            // deposit tokenB
            tx = await tokenBContract.connect(owner).transfer(accountAddress[i].address, parseEther('10000'));
            await checkTransStatus(tx);
            value = await tokenBContract.balanceOf(accountAddress[i].address);
            console.log("balance of account in contractB: ", formatEther(value.toString()));
            expect(value.toString()).equal(parseEther('10000').toString());

            // approve router contract to transfer tokenA
            tx = await tokenAContract.connect(accountAddress[i]).approve(routerContract.address, parseEther('10000'));
            await checkTransStatus(tx);
            console.log(accountAddress[i].address + " approve tokenAContract");

            // approve router contract to transfer tokenB
            tx = await tokenBContract.connect(accountAddress[i]).approve(routerContract.address, parseEther('10000'));
            await checkTransStatus(tx);
            console.log(accountAddress[i].address + " approve tokenBContract")

            // approve router contract to transfer wbnb
            tx = await wbnbContract.connect(accountAddress[i]).approve(routerContract.address, parseEther('10000'));
            await checkTransStatus(tx);
            console.log(accountAddress[i].address + " approve wbnbContract")

            // add liquidity with tokenA, tokenB and WBNB
            let timestamp = Date.parse(new Date());
            timestamp += 300
            tx = await routerContract.connect(accountAddress[i]).addLiquidity(wbnbContract.address,
                tokenAContract.address, parseEther('1'), parseEther('1000'),
                '10000', '10000', accountAddress[i].address, timestamp,
                {gasPrice: gasPrice, gasLimit: "4500000"});
            await checkTransStatus(tx);
            console.log(accountAddress[i].address + " add liquidity routerContract")

            tx = await routerContract.connect(accountAddress[i]).addLiquidity(tokenAContract.address,
                tokenBContract.address, parseEther('1000'), parseEther('1000'),
                '10000', '10000', accountAddress[i].address, timestamp,
                {gasPrice: gasPrice, gasLimit: "4500000"});
            await checkTransStatus(tx);
            console.log(accountAddress[i].address + " add liquidity routerContract")
        }

        // swap
        let timestamp = Date.parse(new Date());
        timestamp += 300
        let value = await tokenBContract.balanceOf(accountAddress[0].address);
        console.log("balance of account in contractB: ", formatEther(value.toString()));
        value = await tokenAContract.balanceOf(accountAddress[0].address);
        console.log("balance of account in contractA: ", formatEther(value.toString()));
        value = await wbnbContract.balanceOf(accountAddress[0].address);
        console.log("balance of account in wbnbContract: ", formatEther(value.toString()));
        let tx = await routerContract.connect(accountAddress[0]).swapExactTokensForTokens(parseUnits('1', 6), '0', bep20TokenContracts,
            accountAddress[1].address, timestamp, {gasPrice: gasPrice, gasLimit: "450000"});
        await checkTransStatus(tx);
    });
});

async function checkTransStatus(tx) {
    let receipt = await tx.wait();
    expect(receipt.status).equal(1);
}