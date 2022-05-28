const path = require('path');
const utils = require('../utils/utils.js');

var Token = artifacts.require('BEP20Token');

module.exports = async function(deployer) {

    var bep20Json = {};
    var addresses = new Array();
    for (var i = 0; i < 16; i++) {
        var tokenName, tokenSymbol;
        if (i < 10) {
            tokenSymbol = 'X0' + i;
        } else {
            tokenSymbol = 'X' + i;
        }
        tokenName = tokenSymbol + ' Token';
        await deployer.deploy(Token, tokenName, tokenSymbol);
        var token = await Token.deployed();
        console.log(token.address);
        addresses.push(token.address);
    }
    bep20Json['addresses'] = addresses;
    var filepath = path.join(path.dirname(__filename), 'bep20.json');
    utils.writeJsonObject(filepath, bep20Json);

};