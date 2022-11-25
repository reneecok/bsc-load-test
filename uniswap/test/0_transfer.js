const { expect } = require('chai');
var chai = require('chai');
const BN = require('bn.js');
chai.use(require('chai-bn')(BN));


describe('transfer bep20 Test', function () {
    beforeEach(async function () {

      VRFConsumer = await ethers.getContractFactory('RandomNumberConsumer');
      vRFConsumer = await VRFConsumer.deploy();
      await vRFConsumer.deployed();
    });

    it('Should make a VRF request', async function () {

      const accounts = await ethers.getSigners()
      const signer = accounts[0]
      const linkTokenContract = new ethers.Contract('0xa36085F69e2889c224210F603D836748e7dC0088',LinkTokenABI, signer)


      var transferTransaction = await linkTokenContract.transfer(vRFConsumer.address,'1000000000000000000')
      await transferTransaction.wait()
      console.log('hash:' + transferTransaction.hash)

      let transaction = await vRFConsumer.getRandomNumber()
      let tx_receipt = await transaction.wait()
      const requestId = tx_receipt.events[2].topics[1]

      await new Promise(resolve => setTimeout(resolve, 60000))

      const result = await vRFConsumer.randomResult()
      console.log('result:' + new ethers.BigNumber.from(result._hex).toString())

      expect((new ethers.BigNumber.from(result._hex).toString())).to.be.a.bignumber.that.is.greaterThan(new ethers.BigNumber.from('0').toString())
    });


  });