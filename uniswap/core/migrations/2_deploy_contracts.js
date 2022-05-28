const path = require('path');
const utils = require('../utils/utils.js');

var Factory = artifacts.require('UniswapV2Factory');

module.exports = async function(deployer, network, addresses) {

    var factoryJson = {};
    await deployer.deploy(Factory, addresses[0]);
    var factory = await Factory.deployed();
    console.log(factory.address);
    factoryJson['address'] = factory.address;
    var filepath = path.join(path.dirname(__filename), 'factory.json');
    utils.writeJsonObject(filepath, factoryJson);

    var basepath = path.resolve(__dirname, '../../');
    filepath = path.join(basepath, 'bep20', 'migrations', 'bep20.json');
    var bep20Json = utils.readJsonObject(filepath);
    var addresses_ = bep20Json['addresses'];

    var from = addresses_.slice(0, addresses_.length / 2);
    var to = addresses_.slice(addresses_.length / 2, addresses_.length);
    console.log(from);
    console.log(to);

    for (var i = 0; i < addresses_.length / 2; i++) {
        var result = await factory.createPair(from[i], to[i]);
        console.log(result.receipt.status);
        var pair = await factory.getPair(from[i], to[i]);
        console.log(pair);
    }

};