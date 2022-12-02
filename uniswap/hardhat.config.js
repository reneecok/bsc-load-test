require('dotenv').config()
require("@nomiclabs/hardhat-ethers");
require('solidity-coverage')


const RPC_URL = process.env.RPC_URL
const PRIVATE_KEY = process.env.PRIVATE_KEY
module.exports = {
  defaultNetwork: "hardhat",
  networks: {
    default: {
      url: RPC_URL,
      accounts: [PRIVATE_KEY]
    }
  },
  solidity: {
    compilers: [
    {version: "0.8.0"},
    {version: "0.8.7"},
    {version: "0.6.6",
    settings: {
      optimizer: {
        enabled: true,
        runs: 1
      }
    }},
    {version:"0.6.10"},
    {version:"0.6.0"},
    {version:"0.5.16"}
  ],
    settings: {
      optimizer: {
        enabled: true,
        runs: 1
      }
    }
  },
  paths: {
    sources: "./contracts",
    tests: "./test",
    cache: "./cache",
    artifacts: "./artifacts"
  },
  mocha: {
    timeout: 3000000
  }
}