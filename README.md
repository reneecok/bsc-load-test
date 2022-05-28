docker run -v /Users/user/Downloads/nodereal/bsc-load-test/uniswap/bep20/contracts:/sources ethereum/solc:0.5.16 --abi -o /sources /sources/bep20.sol --overwrite

docker run -v /Users/user/Downloads/nodereal/bsc-load-test/uniswap/bep20/contracts:/sources ethereum/solc:0.5.16 --bin -o /sources /sources/bep20.sol --overwrite

/Users/user/Downloads/go-ethereum-1.10.3/build/bin/abigen --abi=./BEP20Token.abi --bin=./BEP2EToken.bin --pkg=bep20 --out=BEP20Token.go

docker run -v /Users/user/Downloads/nodereal/bsc-load-test/uniswap/core/contracts:/sources ethereum/solc:0.6.6 --abi -o /sources /sources/interfaces/IUniswapV2Factory.sol --overwrite

/Users/user/Downloads/go-ethereum-1.10.3/build/bin/abigen --abi=./IUniswapV2Factory.abi --pkg=V2factory --out=UniswapV2Factory.go

docker run -v /Users/user/Downloads/nodereal/bsc-load-test/uniswap/periphery/contracts:/sources ethereum/solc:0.6.6 --abi -o /sources /sources/interfaces/IUniswapV2Router01.sol --overwrite

/Users/user/Downloads/go-ethereum-1.10.3/build/bin/abigen --abi=./IUniswapV2Router01.abi --pkg=v2router --out=UniswapV2Router.go

docker run -v /Users/user/Downloads/nodereal/bsc-load-test/uniswap/periphery/contracts:/sources ethereum/solc:0.6.6 --abi -o /sources /sources/WBNB.sol --overwrite

/Users/user/Downloads/go-ethereum-1.10.3/build/bin/abigen --abi=./WBNB.abi --pkg=wbnb --out=WBNB.go

go build -o build/driver driver.go

//****once only****

-randTestAcc=true -hexkeyfile=/Users/user/Downloads/nodereal/bsc-load-test/wallets/hexkey_active_0.csv -usersCreated=100000
-randTestAcc=true -hexkeyfile=/Users/user/Downloads/nodereal/bsc-load-test/wallets/hexkey_active_1.csv -usersCreated=250000

-endpoints="https://data-seed-prebsc-1-s2.binance.org:8545" -roothexkey=5f4e1f061c905b0e8c5913d3683f2f3353bf2dfb0c91713d7f293d9a597b0125 -roothexaddr=0x89E73303049EE32919903c09E8DE5629b84f59EB -bep20Hex=0xd929Aa344e14E465B3a6468742E47AE3775882Fb,0xb4A9E31E341DbDEcCe6eE4b3C6CFa43233151165,0xF1FB747002Df5A0769cBf70757F1D6bd0EbEA737,0xF820eA5208494848957Be979998a904335Eb8D4F,0xF33a7651340215151dE9C1168699d5501b1EF36a,0x857D855A3Bff368ddE9b95795B92941eDeB87015,0x6e75Ce6c0Ea5375Ce78fe5D87074c9bD3DCc6D51,0x9848BcECB273570964243691C8D02a5ffa2B9b64,0xdFFc047542430674f982684Fd09974c88CAEFb97,0xBb9fA3876ade1B7e57B05cFaf99a26eAefA52Bd8,0xE0179C1ad927Fc698f725dd92Ebc96735dfC912D,0x38655FE5183406B12d3a93c93C3e98C874DD99fd,0xf7481BaE5b4CC928a2CEF53729D08215bBADC15E,0x35e171ea29bcddA8f4e539370b9595cfb4a3FeE1,0x9a290b46b8E8bCfAc2880d067d0Bee25941b0740,0x8ad4Ac8A0BBCD1b303086b239a9e2cB67452EAd4 -wbnbHex=0xa841374bb918D4E16eD414ae71800efC211d60d6 -uniswapFactoryHex=0xB1284390771375b14Ca75D80c9626D6Dfd859140 -uniswapRouterHex=0x67d0B264946e47d99cA6BD9A33242c90b917a212

// init

-endpoints="https://data-seed-prebsc-1-s2.binance.org:8545" -roothexkey=5f4e1f061c905b0e8c5913d3683f2f3353bf2dfb0c91713d7f293d9a597b0125 -roothexaddr=0x89E73303049EE32919903c09E8DE5629b84f59EB -bep20Hex=0xd929Aa344e14E465B3a6468742E47AE3775882Fb,0xb4A9E31E341DbDEcCe6eE4b3C6CFa43233151165,0xF1FB747002Df5A0769cBf70757F1D6bd0EbEA737,0xF820eA5208494848957Be979998a904335Eb8D4F,0xF33a7651340215151dE9C1168699d5501b1EF36a,0x857D855A3Bff368ddE9b95795B92941eDeB87015,0x6e75Ce6c0Ea5375Ce78fe5D87074c9bD3DCc6D51,0x9848BcECB273570964243691C8D02a5ffa2B9b64,0xdFFc047542430674f982684Fd09974c88CAEFb97,0xBb9fA3876ade1B7e57B05cFaf99a26eAefA52Bd8,0xE0179C1ad927Fc698f725dd92Ebc96735dfC912D,0x38655FE5183406B12d3a93c93C3e98C874DD99fd,0xf7481BaE5b4CC928a2CEF53729D08215bBADC15E,0x35e171ea29bcddA8f4e539370b9595cfb4a3FeE1,0x9a290b46b8E8bCfAc2880d067d0Bee25941b0740,0x8ad4Ac8A0BBCD1b303086b239a9e2cB67452EAd4 -wbnbHex=0xa841374bb918D4E16eD414ae71800efC211d60d6 -uniswapFactoryHex=0xB1284390771375b14Ca75D80c9626D6Dfd859140 -uniswapRouterHex=0x67d0B264946e47d99cA6BD9A33242c90b917a212 -initTestAcc -tps=1 -hexkeyfile=/Users/user/Downloads/nodereal/bsc-load-test/wallets/hexkey_0.csv -usersLoaded=16

// test

-endpoints="https://data-seed-prebsc-1-s2.binance.org:8545" -roothexkey=5f4e1f061c905b0e8c5913d3683f2f3353bf2dfb0c91713d7f293d9a597b0125 -roothexaddr=0x89E73303049EE32919903c09E8DE5629b84f59EB -bep20Hex=0xd929Aa344e14E465B3a6468742E47AE3775882Fb,0xb4A9E31E341DbDEcCe6eE4b3C6CFa43233151165,0xF1FB747002Df5A0769cBf70757F1D6bd0EbEA737,0xF820eA5208494848957Be979998a904335Eb8D4F,0xF33a7651340215151dE9C1168699d5501b1EF36a,0x857D855A3Bff368ddE9b95795B92941eDeB87015,0x6e75Ce6c0Ea5375Ce78fe5D87074c9bD3DCc6D51,0x9848BcECB273570964243691C8D02a5ffa2B9b64,0xdFFc047542430674f982684Fd09974c88CAEFb97,0xBb9fA3876ade1B7e57B05cFaf99a26eAefA52Bd8,0xE0179C1ad927Fc698f725dd92Ebc96735dfC912D,0x38655FE5183406B12d3a93c93C3e98C874DD99fd,0xf7481BaE5b4CC928a2CEF53729D08215bBADC15E,0x35e171ea29bcddA8f4e539370b9595cfb4a3FeE1,0x9a290b46b8E8bCfAc2880d067d0Bee25941b0740,0x8ad4Ac8A0BBCD1b303086b239a9e2cB67452EAd4 -wbnbHex=0xa841374bb918D4E16eD414ae71800efC211d60d6 -uniswapFactoryHex=0xB1284390771375b14Ca75D80c9626D6Dfd859140 -uniswapRouterHex=0x67d0B264946e47d99cA6BD9A33242c90b917a212 -runTestAcc -tps=1 -sec=10 -hexkeyfile=/Users/user/Downloads/nodereal/bsc-load-test/wallets/hexkey_0.csv -usersLoaded=16


// query

-endpoints="https://data-seed-prebsc-1-s2.binance.org:8545" -roothexkey=5f4e1f061c905b0e8c5913d3683f2f3353bf2dfb0c91713d7f293d9a597b0125 -roothexaddr=0x89E73303049EE32919903c09E8DE5629b84f59EB -queryBlocks -blockNumS=19697251 -blockNumE=200000000
