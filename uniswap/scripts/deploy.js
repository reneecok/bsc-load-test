// import { ethers } from "hardhat";
const { ethers } = require('hardhat');
const path = require('path');

const utils = require('./utils.js');

async function main() {
    const [owner] = await ethers.getSigners();
    var contractAddress = {};
    console.log('owner: ' + owner.address)
    // We get the contract to deploy bep20
    var Token = await ethers.getContractFactory('BEP20Token');
    var Factory = await ethers.getContractFactory('UniswapV2Factory');
    var WBNB = await ethers.getContractFactory('WBNB');
    var Router = await ethers.getContractFactory('UniswapV2Router02');

    var addresses = new Array();
    for (var i = 0; i < 16; i++) {
        var tokenName, tokenSymbol;
        if (i < 10) {
            tokenSymbol = 'X0' + i;
        } else {
            tokenSymbol = 'X' + i;
        }
        tokenName = tokenSymbol + ' Token';
        var token = await Token.deploy(tokenName, tokenSymbol);
        await token.deployed();
        console.log(tokenName + ": " + token.address);
        addresses.push(token.address);
    }
    contractAddress['bep20'] = addresses;
    console.log("==== bep20 deployed======");
    let printBep20Address = addresses[0]
    for (let i = 1; i < addresses.length; i++) {
        printBep20Address += "," + addresses[i]
    }
    console.log('factory: ' + printBep20Address);

    var factory = await Factory.deploy(owner.address);
    await factory.deployed();
    console.log('factory: '+factory.address);
    
    let pairInitHash = await factory.getPairInitHash();
    console.log("pairInitHash:", pairInitHash.toString())
    contractAddress['factory'] = factory.address;
    contractAddress['pairInitHash'] = pairInitHash;


    var addresses_ = contractAddress['bep20'];

    var from = addresses_.slice(0, addresses_.length / 2);
    var to = addresses_.slice(addresses_.length / 2, addresses_.length);
    console.log('from: ' + from);
    console.log('to: ' + to);

    let wbnb = await WBNB.deploy();
    await wbnb.deployed();
    console.log('wbnb: '+ wbnb.address);
    contractAddress['WBNB'] = wbnb.address

    for (var i = 0; i < addresses_.length / 2; i++) {
        let transaction = await factory.createPair(from[i], to[i]);
        let tx_receipt = await transaction.wait()
        console.log('createPair 0  status: ' + tx_receipt.status);

        let transaction2 = await factory.createPair(from[i], wbnb.address);
        let tx_receipt2 = await transaction2.wait()
        console.log('createPair 1 success tx status: ' + tx_receipt2.status);

        let transaction3 = await factory.createPair(wbnb.address, to[i]);
        let tx_receipt3 = await transaction3.wait()
        console.log('createPair 2 success tx status: ' + tx_receipt3.status);
        
        var pair = await factory.getPair(from[i], to[i]);
        console.log('index: ' + i + ' pair: ' + pair);
    }


    var router = await Router.deploy(factory.address, wbnb.address)
    await router.deployed();
    console.log('router: '+ router.address);
    contractAddress['Router'] = router.address

    var filepath = path.join(path.dirname(__filename), 'contracts.json');
    utils.writeJsonObject(filepath, contractAddress);
  }

  main()
    .then(() => process.exit(0))
    .catch((error) => {
      console.error(error);
      process.exit(1);
    });