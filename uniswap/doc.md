1. Create .env file from .env.example file
2. Set Up The New Network Params
   1. edit rpc URL and owner private key in .env file, change the default number of contract would be deployed
   2. make sure that there are enough tokens in the owner account  
3. Deploy Uniswap Contracts To The New Network
   ```shell
   cd uniswap
   nvm install 18.0.0
   nvm use 18.0.0
   yarn install
   npx hardhat run --network default scripts/deploy.js
   ```
4. Init Accounts
   1. for each network, accounts initialization only needs to be done once with each csv file in wallets/hexkey.csv
   2. change the number of accounts with param usersLoaded 
   3. replace uniswapRouterHex, wbnbHex, bep20Hex and uniswapFactoryHex params after the new deployment finished
   4. replace endpoints, roothexaddr and roothexkey as you needed
   ```shell
   nohup ./driver -endpoints=http://netmarble-bas-dataseed:8545 -roothexkey=5f4e1f061c905b0e8c5913d3683f2f3353bf2dfb0c91713d7f293d9a597b0125 -roothexaddr=0x89E73303049EE32919903c09E8DE5629b84f59EB -bep20Hex="0x05A4A25cd95e7f442034204F02F500550dcd0E66,0x3B6e0aD8dDA16dEfB7cdF23AFa46691Da52e120E,0xADfeFEc44Fd27381DC6AfC95023300C7A3A2feB9,0x3722e43b6A8f762c3dd2395df7dbE35921F45245,0x1187cF107363d074012807447E1b639c893b75c1,0xec048627004Db8eB82e2E6F41946237858936c2B,0x951964e979BbFAc0C0b2989463412a497710517F,0x84eA9c32524e6498604C67aC6a2080bf172560df,0x2FcC5444368aac7a644f82D3cc7b043b9324DC0f,0x887E8D6dAB7D904C0Ffff89cfFd4080C2f5404ED,0xff85963aB463cd6c930290F3805CEF5b37783444,0x837297f7927CA202213Ee3BDf86E8434994820b2,0xE59FeCb915206bE547E5DaC2D8D31D05fCDCDd9D,0xC89fc05E0AAa0401b9C3b806802423EA39598Dd0,0xC83bf511f3194b29805f911c1bcbDB9f70376339,0xCcABc2D10a5a18409e4c54D6E14F42B873A1d0c3" -wbnbHex=0xadE1dD6E4218DFdA880e96695c77123916e8Ac22 -uniswapFactoryHex=0x933eA9e5D21c48022a31582eBA3801048F6427B7 -uniswapRouterHex=0xBA5606edDe168631d17C38aD9246b2727D773E08 -initTestAcc -tps=80  -hexkeyfile=./wallets/hexkey_0.csv -usersLoaded=50000 > n.log &

   nohup ./driver -endpoints=https://fncy-testnet-seed.fncy.world -roothexkey=5f4e1f061c905b0e8c5913d3683f2f3353bf2dfb0c91713d7f293d9a597b0125 -roothexaddr=0x89E73303049EE32919903c09E8DE5629b84f59EB -bep20Hex="0x8D929A0dC23AFF814117B4e7D783018E11fA4B65,0x654acEC030bf709B783E0aA80432F971Bf399a63" -uniswapRouterHex=0x2E12A3d0FEC1d8D8a0172c67ca404844F029B49f -wbnbHex=0x6AcC1d1eAACD4B9D4c4F767fe92Be2F947904E21 -uniswapFactoryHex=0xE3b43662EA55F6294298cCFECa3BdDB8651b05d9 -erc721Hex=0x3310951A1f558d2570258aa38b0228F830c82460 -erc1155Hex=0xAFb372aDFe0d42127522ba457268E25f6EA88e93 -initTestAcc -tps=2 -sec=10 -hexkeyfile=../wallets/hexkey_0.csv -usersLoaded=100 -slaveUserHexkeyFile=../wallets/hexkey_1.csv -slaveUserLoaded=10 > slave.log &

   ```
5. Run Test
   1. change expected test time with param sec
   2. check on the monitor of the network and adjust param tps
   ```shell
   nohup ./driver -endpoints=http://10.179.215.133:8545 -roothexkey=5f4e1f061c905b0e8c5913d3683f2f3353bf2dfb0c91713d7f293d9a597b0125 -roothexaddr=0x89E73303049EE32919903c09E8DE5629b84f59EB -bep20Hex="0x05A4A25cd95e7f442034204F02F500550dcd0E66,0x3B6e0aD8dDA16dEfB7cdF23AFa46691Da52e120E,0xADfeFEc44Fd27381DC6AfC95023300C7A3A2feB9,0x3722e43b6A8f762c3dd2395df7dbE35921F45245,0x1187cF107363d074012807447E1b639c893b75c1,0xec048627004Db8eB82e2E6F41946237858936c2B,0x951964e979BbFAc0C0b2989463412a497710517F,0x84eA9c32524e6498604C67aC6a2080bf172560df,0x2FcC5444368aac7a644f82D3cc7b043b9324DC0f,0x887E8D6dAB7D904C0Ffff89cfFd4080C2f5404ED,0xff85963aB463cd6c930290F3805CEF5b37783444,0x837297f7927CA202213Ee3BDf86E8434994820b2,0xE59FeCb915206bE547E5DaC2D8D31D05fCDCDd9D,0xC89fc05E0AAa0401b9C3b806802423EA39598Dd0,0xC83bf511f3194b29805f911c1bcbDB9f70376339,0xCcABc2D10a5a18409e4c54D6E14F42B873A1d0c3" -wbnbHex=0xadE1dD6E4218DFdA880e96695c77123916e8Ac22 -uniswapFactoryHex=0x933eA9e5D21c48022a31582eBA3801048F6427B7 -uniswapRouterHex=0xBA5606edDe168631d17C38aD9246b2727D773E08   -runTestAcc -tps=150 -sec=360000 -hexkeyfile=./wallets/hexkey_0.csv -usersLoaded=50000 > n.log &
   nohup ./driver -endpoints=https://fncy-testnet-seed.fncy.world -roothexkey=5f4e1f061c905b0e8c5913d3683f2f3353bf2dfb0c91713d7f293d9a597b0125 -roothexaddr=0x89E73303049EE32919903c09E8DE5629b84f59EB -bep20Hex="0x8D929A0dC23AFF814117B4e7D783018E11fA4B65,0x654acEC030bf709B783E0aA80432F971Bf399a63" -uniswapRouterHex=0x2E12A3d0FEC1d8D8a0172c67ca404844F029B49f -wbnbHex=0x6AcC1d1eAACD4B9D4c4F767fe92Be2F947904E21 -uniswapFactoryHex=0xE3b43662EA55F6294298cCFECa3BdDB8651b05d9 -erc721Hex=0x3310951A1f558d2570258aa38b0228F830c82460 -erc1155Hex=0xAFb372aDFe0d42127522ba457268E25f6EA88e93 -runTestAcc -tps=1 -sec=600 -hexkeyfile=../wallets/hexkey_0.csv -usersLoaded=100  -slaveUserHexkeyFile=../wallets/hexkey_1.csv -slaveUserLoaded=10 > nft1155_new_re.log &
   ```
6. Other Functions
   check on block info whether the network is working 
   ```shell
   ./driver -endpoints=http://netmarble-bas-dataseed:8545  -roothexkey=5f4e1f061c905b0e8c5913d3683f2f3353bf2dfb0c91713d7f293d9a597b0125 -roothexaddr=0x89E73303049EE32919903c09E8DE5629b84f59EB -queryBlocks -blockNumS=459010 -blockNumE=459112
   ```
7. Run Unit Test With Hardhat
   1. run deployment of bep20 token and transfer
      ```shell
      npx hardhat test --network default test/0_transfer.js
      OR 
      npx hardhat test --network default  --grep "deploy and transfer bep20 token"
      ```
   2. run deployment of uniswap contract and swap token
      ```shell
      npx hardhat test --network default test/1_swap.js
      OR
      npx hardhat test --network default  --grep "swap tokens"
      ```

