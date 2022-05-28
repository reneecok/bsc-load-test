uniswap deployment on BSC testnet

https://docs.binance.org/smart-chain/developer/deploy/truffle.html
https://docs.binance.org/smart-chain/developer/deploy/truffle-verify.html

npm install -g truffle
npm install @truffle/hdwallet-provider
npm install -D truffle-plugin-verify

truffle migrate --network testnet --reset

bep20: deploy dep20 tokens

[
  '0x02Bdb6a65e7356e69807A987C94A42DEfB45f1f5',
  '0x157A7C9205b38B400cB9e9f7d2043ab01CE27c4E',
  '0x497cD3E41dbB984f5944167729999BDf434C5C50',
  '0xec78712527cB30C84F08CcB8fE87dB016c02a152',
  '0xB6F5bd1D4f46f0dF207D2be0f3853f0F300B980b',
  '0x0e80833BF081C43f86f1Ec24280975C4CAB9470b',
  '0xa7aF12655C779b0b0ff3308B059ff5f6F364999b',
  '0x9E822bB990123f0Fd66f4A13EEF5EF35831Ee9C7',
  '0x05362c0f966c77D74f4ccAC440E287D50614babC',
  '0x27592c03182dCD55f006622a5De725322534b4A2',
  '0x3f931c1991E3081666288560873150583f749756',
  '0x2259Ce160dd8f5C8CCdd4Ec5728d1f4a96269406',
  '0xb6B4F9849567823A8dc6Ce3009F37209a82773A7',
  '0x3e2a118f5216a4D196756721921977B8a97fF99A',
  '0x4FbeB966c7D8870Fc7AB83F605aE1F968832e70a',
  '0xB6a3b7e5Cb237bb7BFCc0F22D60aB349099385e5'
]

core: deploy uniswap factory

0xD3b8AbC44274ebc1E737E23eaF07C41d05c156F2

periphery: deploy uniswap router and wbnb

truffle run verify UniswapV2Factory@0xB1284390771375b14Ca75D80c9626D6Dfd859140 --network testnet
truffle run verify WBNB@0xa841374bb918D4E16eD414ae71800efC211d60d6 --network testnet
truffle run verify UniswapV2Router02@0x67d0B264946e47d99cA6BD9A33242c90b917a212 --network testnet

[important]

UniswapV2Router02.sol:61 addLiquidity() -> UniswapV2Router02.sol:45 getReserves() -> execution reverted

"(uint reserve0, uint reserve1,) = IUniswapV2Pair(pairFor(factory, tokenA, tokenB)).getReserves();"

bsc-loadtest/uniswap/periphery/contracts/libraries/UniswapV2Library.sol

    function pairFor(address factory, address tokenA, address tokenB) internal pure returns (address pair) {
        (address token0, address token1) = sortTokens(tokenA, tokenB);
        pair = address(uint(keccak256(abi.encodePacked(
                hex'ff',
                factory,
                keccak256(abi.encodePacked(token0, token1)),
                hex'03f6509a2bb88d26dc77ecc6fc204e95089e30cb99667b85e653280b735767c8' // init code hash
            ))));
    }

get the init code hash using the method below, update the above hard coded one, redeploy factory and router

bsc-loadtest/uniswap/core/contracts/UniswapV2Factory.sol

    function getPairInitHash() public pure returns(bytes32){
        bytes memory bytecode = type(UniswapV2Pair).creationCode;
        return keccak256(abi.encodePacked(bytecode));
    }
