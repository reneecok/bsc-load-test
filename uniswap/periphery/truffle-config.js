const HDWalletProvider = require('@truffle/hdwallet-provider');
const fs = require('fs');
const mnemonic = fs.readFileSync(".secret").toString().trim();

module.exports = {

  plugins: [
    'truffle-plugin-verify'
  ],
  api_keys: {
    bscscan: 'E71HRW4EJR9TGVCTSC2MCQ4DM78GQG9EU1'
  },

  networks: {
    development: {
      host: "127.0.0.1",     // Localhost (default: none)
      port: 8545,            // Standard BSC port (default: none)
      network_id: "*",       // Any network (default: none)
    },
    qa: {
      provider: () => new HDWalletProvider(mnemonic, `http://172.22.41.197:8545`),
      network_id: 96,
      confirmations: 10,
      timeoutBlocks: 200,
      skipDryRun: true
    },
    testnet: {
      provider: () => new HDWalletProvider(mnemonic, `https://data-seed-prebsc-1-s1.binance.org:8545`),
      network_id: 97,
      confirmations: 10,
      timeoutBlocks: 200,
      skipDryRun: true
    },
    bsc: {
      provider: () => new HDWalletProvider(mnemonic, `https://bsc-dataseed1.binance.org`),
      network_id: 56,
      confirmations: 10,
      timeoutBlocks: 200,
      skipDryRun: true
    },
  },

  // Set default mocha options here, use special reporters etc.
  mocha: {
    // timeout: 100000
  },

  // Configure your compilers
  compilers: {
    solc: {
      version: "0.6.6", // A version or constraint - Ex. "^0.5.0"
      settings: {
        optimizer: {
          enabled: true,
          runs: 1500   // Optimize for how many times you intend to run the code
        }
      }
    }
  }
}