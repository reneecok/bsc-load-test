1. Create uniswap/.env file from uniswap/.env.example file
2. Set Up the New Network Params
   1. edit rpc URL and owner private key in .env file, change the default number of erc20 contract would be deployed
   2. make sure that there are enough tokens in the owner account  
3. Deploy Uniswap Contracts to the New Network
   ```shell
   cd uniswap
   nvm install 18.0.0
   nvm use 18.0.0
   yarn install
   npx hardhat run --network default scripts/deploy.js
   ```
4. Init Accounts
   1. for each network, accounts initialization only needs to be done once with each csv file in wallets/hexkey.csv
   2. change the number of accounts UsersLoaded in config.yml
   3. replace endpoints, roothexaddr and roothexkey as you needed
   4. the config.yml file will be updated automatically, you can check the contract info and chain info
   5. set up tps in command line, slaveUserHexkeyFile and slaveUserLoaded params in config.yml 
   ```shell
      # tps 400 means 
      # 1. init BNB and BEP20  200 account/second (10w account need 500s);
      # 2. init uniswap and nft 40 account/second (10w account need use about 10w/40 = 2500s)
      # 10w account init need 3000s (2500s+500s )
   nohup ./driver -initTestAcc -tps=400 > n.log & 

   ```
5. Run Test
   1. change expected test time with sec params and tps params in command line or config.yml
   2. change slaveUserHexkeyFile and slaveUserLoaded params in config.yml
   3. check on the monitor of the network and adjust param tps
   ```shell
   nohup ./driver -runTestAcc -tps=150 -sec=360000 > n.log &
   ```
6. Other Functions
   check on block info whether the network is working 
   ```shell
   ./driver -queryBlocks -blockNumS=459010 -blockNumE=459112
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

