// import { ethers } from "hardhat";
require('dotenv').config('../.env')
const {ethers} = require('hardhat');
const path = require('path');
const fs = require('fs');
const {parseDocument} = require('yaml');

const utils = require('./utils.js');
const { providers } = require('ethers');
const numOfContract = process.env.NumberOfContract

async function main() {

    const [owner] = await ethers.getSigners();
    chainId = await owner.getChainId()
    let contractAddress = {};
    console.log('owner: ' + owner.address)
    // We get the contract to deploy bep20
    let Token = await ethers.getContractFactory('BEP20Token');
    let Factory = await ethers.getContractFactory('UniswapV2Factory');
    let WBNB = await ethers.getContractFactory('WBNB');
    let Router = await ethers.getContractFactory('UniswapV2Router02');
    let Nft721 = await ethers.getContractFactory('MyToken')
    let Nft1155 = await ethers.getContractFactory('TERC1155')
    let feedata = await ethers.provider.getFeeData();
    console.log('feedata: ',feedata)

    let gasPrice = await ethers.provider.getGasPrice();
    console.log('gasPrice: ',gasPrice.toString())
    let addresses = [];
    for (let i = 0; i < numOfContract; i++) {
        let tokenName, tokenSymbol;
        if (i < 10) {
            tokenSymbol = 'X0' + i;
        } else {
            tokenSymbol = 'X' + i;
        }
        tokenName = tokenSymbol + ' Token';
        let token = await Token.deploy(tokenName, tokenSymbol, {gasPrice: gasPrice.toString()});
        await token.deployed();
        console.log(tokenName + ": " + token.address);
        addresses.push(token.address);
    }
    contractAddress['bep20'] = addresses;
    console.log("==== bep20 deployed======");
    console.log('bep20: ' + addresses);

    let factory = await Factory.deploy(owner.address);
    await factory.deployed();
    console.log('factory: ' + factory.address);

    let pairInitHash = await factory.getPairInitHash();
    console.log("pairInitHash:", pairInitHash.toString())
    contractAddress['factory'] = factory.address;
    contractAddress['pairInitHash'] = pairInitHash;


    let addresses_ = contractAddress['bep20'];

    let from = addresses_.slice(0, addresses_.length / 2);
    let to = addresses_.slice(addresses_.length / 2, addresses_.length);
    console.log('from: ' + from);
    console.log('to: ' + to);

    let wbnb = await WBNB.deploy();
    await wbnb.deployed();
    console.log('wbnb: ' + wbnb.address);
    contractAddress['WBNB'] = wbnb.address

    for (let i = 0; i < addresses_.length / 2; i++) {

        let transaction = await factory.createPair(from[i], to[i]);
        let tx_receipt = await transaction.wait()
        console.log('pair: ' , from[i], to[i],' status: ' + tx_receipt.status);

        let pair0 = await factory.getPair(from[i], to[i]);
        console.log('index: ' + i + ' pair0: ' + pair0)

        let transaction2 = await factory.createPair(from[i], wbnb.address);
        let tx_receipt2 = await transaction2.wait()
        console.log('pair: ' , from[i], wbnb.address,' status: '  + tx_receipt2.status);
        let pair1 = await factory.getPair(from[i], wbnb.address);
        console.log('index: ' + i + ' pair1: ' + pair1)

        let transaction3 = await factory.createPair(to[i], wbnb.address);
        let tx_receipt3 = await transaction3.wait()
        console.log('pair: ' , to[i], wbnb.address,' status: '  + tx_receipt3.status);

        let pair2 = await factory.getPair(to[i], wbnb.address);
        console.log('index: ' + i + ' pair2: ' + pair2)

    }


    let router = await Router.deploy(factory.address, wbnb.address)
    await router.deployed();
    console.log('router: ' + router.address);
    contractAddress['Router'] = router.address

    let nft721 = await Nft721.deploy()
    await nft721.deployed()
    console.log('nft721: ' + nft721.address);
    contractAddress['Nft721'] = nft721.address

    let nft1155 = await Nft1155.deploy()
    await nft1155.deployed()
    console.log('nft1155: ' + nft1155.address);
    contractAddress['Nft1155'] = nft1155.address


    // print out all params
    let ben20ContractAddress = contractAddress["bep20"];
    let printBep20Address = ben20ContractAddress[0];
    for (let i = 1; i < ben20ContractAddress.length; i++) {
        printBep20Address += "," + ben20ContractAddress[i];
    }
    let routerContractAddress = contractAddress["Router"];
    let wbnbContractAddress = contractAddress['WBNB'];
    let factoryContractAddress = contractAddress['factory'];
    let nft721ContractAddress = contractAddress['Nft721'];
    let nft1155ContractAddress = contractAddress['Nft1155'];

    console.log("\nCopy Command Line Params From Here: \n")
    console.log("-bep20Hex=\"" + printBep20Address + "\"" + " " + "-uniswapRouterHex=" + routerContractAddress + " " +
        "-wbnbHex=" + wbnbContractAddress + " " + "-uniswapFactoryHex=" + factoryContractAddress + " -erc721Hex=" + nft721ContractAddress + " " +
        "-erc1155Hex=" + nft1155ContractAddress)

    let file = fs.readFileSync('../config.yml', 'utf-8');
    const origin = parseDocument(file);
    origin.set("Bep20Hex", printBep20Address)
    origin.set("WbnbHex", wbnbContractAddress)
    origin.set("UniswapFactoryHex", factoryContractAddress)
    origin.set("UniswapRouterHex", routerContractAddress)
    origin.set("Erc721Hex", nft721ContractAddress)
    origin.set("Erc1155Hex", nft1155ContractAddress)
    origin.set("ChainId", chainId)
    origin.set("Endpoints", process.env.RPC_URL)

    fs.writeFileSync('../config.yml', origin.toString());
}

main()
    .then(() => process.exit(0))
    .catch((error) => {
        console.error(error);
        process.exit(1);
    });