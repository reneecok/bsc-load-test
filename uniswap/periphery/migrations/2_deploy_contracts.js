const path = require('path');
const utils = require('../utils/utils.js');

var WBNB = artifacts.require('WBNB');
var Router = artifacts.require('UniswapV2Router02');

module.exports = async function(deployer, network) {

    var basepath = path.resolve(__dirname, '../../');
    var filepath = path.join(basepath, 'core', 'migrations', 'factory.json');
    var factoryJson = utils.readJsonObject(filepath);

    var wbnb;
    if (network == 'mainnet') {
        wbnb = await WBNB.at('0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c');
    } else {
        await deployer.deploy(WBNB);
        wbnb = await WBNB.deployed();
    }

    await deployer.deploy(Router, factoryJson['address'], wbnb.address);
    var router = await Router.deployed();
    console.log('router: ' + router.address)
    var wbnbJson = {};
    filepath = path.join(path.dirname(__filename), 'wbnb.json');
    wbnbJson['address'] = wbnb.address;
    utils.writeJsonObject(filepath, wbnbJson);

    var routerJson = {};
    filepath = path.join(path.dirname(__filename), 'router.json');
    routerJson['address'] = router.address;
    utils.writeJsonObject(filepath, routerJson);

};