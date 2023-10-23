package utils

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math"
	"math/big"
	"time"

	"bsc-load-test/contracts/V2factory"
	"bsc-load-test/contracts/V2router"
	"bsc-load-test/contracts/bep20"
	"bsc-load-test/contracts/erc1155"
	"bsc-load-test/contracts/erc721"
	"bsc-load-test/contracts/wbnb"
	"bsc-load-test/log"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var contracts Contract

type Contract struct {
	V2RouterInstance *V2router.V2router
	WbnbInstance     *wbnb.Wbnb
	FactoryInstance  *V2factory.V2factory
	Erc721Instance   *erc721.Erc721
	Erc1155Instance  *erc1155.Erc1155
	Bep20InstanceMap map[string]*bep20.Bep20
}

func InitContacts(root *ExtAcc) {
	var err error
	if T_cfg.UniswapRouterAddr.String() != "" {
		if contracts.V2RouterInstance, err = V2router.NewV2router(T_cfg.UniswapRouterAddr, root.Client); err != nil {
			log.Println("error: create V2router instance failed")
		}
	}
	if T_cfg.WbnbAddr.String() != "" {
		if contracts.WbnbInstance, err = wbnb.NewWbnb(T_cfg.WbnbAddr, root.Client); err != nil {
			log.Println("error: create wbnb instance failed")
		}
	}
	if T_cfg.UniswapFactoryAddr.String() != "" {
		if contracts.FactoryInstance, err = V2factory.NewV2factory(T_cfg.UniswapFactoryAddr, root.Client); err != nil {
			log.Println("error: create factory instance failed")
		}
	}
	if T_cfg.Erc721Addr.String() != "" {
		if contracts.Erc721Instance, err = erc721.NewErc721(T_cfg.Erc721Addr, root.Client); err != nil {
			log.Println("error: create erc721 instance failed")
		}
	}
	if T_cfg.Erc1155Addr.String() != "" {
		if contracts.Erc1155Instance, err = erc1155.NewErc1155(T_cfg.Erc1155Addr, root.Client); err != nil {
			log.Println("error: create erc1155 instance failed")
		}
	}
	instanceMap := make(map[string]*bep20.Bep20)
	if len(T_cfg.Bep20AddrsA) > 0 && len(T_cfg.Bep20AddrsB) > 0 {
		// add bep20 contract and wbnb instance
		bep20Addrs := append(T_cfg.Bep20AddrsA, T_cfg.Bep20AddrsB...)
		bep20Addrs = append(bep20Addrs, T_cfg.WbnbAddr)
		for _, address := range bep20Addrs {
			instance, err := bep20.NewBep20(address, root.Client)
			if err != nil {
				log.Println("error: create bep20 instance failed")
			}
			instanceMap[address.String()] = instance
		}

		// add contract pair instance: A-b, wbnb-b
		for i, address := range T_cfg.Bep20AddrsA {
			pair, err := root.GetPair(&address, &T_cfg.Bep20AddrsB[i])
			if err != nil {
				log.Println("error: get pair:", err)
				continue
			}
			instance, err := bep20.NewBep20(*pair, root.Client)
			if err != nil {
				log.Println("error: create bep20 instance failed")
			}
			instanceMap[pair.String()] = instance

			pair, err = root.GetPair(&T_cfg.WbnbAddr, &address)
			if err != nil {
				log.Println("error: get pair:", err)
				continue
			}
			instance, err = bep20.NewBep20(*pair, root.Client)
			if err != nil {
				log.Println("error: create bep20 instance failed")
			}
			instanceMap[pair.String()] = instance
		}
		contracts.Bep20InstanceMap = instanceMap
	}
	log.Println(contracts.Bep20InstanceMap)
}

type ExtAcc struct {
	Client *ethclient.Client
	Key    *ecdsa.PrivateKey
	Addr   *common.Address
}

func NewExtAcc(client *ethclient.Client, hexkey string, hexaddr string) (*ExtAcc, error) {
	key, err := crypto.HexToECDSA(hexkey)
	if err != nil {
		return nil, err
	}
	addr := common.HexToAddress(hexaddr)
	ex := &ExtAcc{
		Client: client,
		Key:    key,
		Addr:   &addr,
	}
	return ex, nil
}

func (ea *ExtAcc) GetBNBBalance() (*big.Int, error) {
	ctx := context.Background()
	balance, err := ea.Client.BalanceAt(ctx, *ea.Addr, nil)
	if err != nil {
		return nil, err
	}
	fbal := new(big.Float)
	fbal.SetString(balance.String())
	value := new(big.Float).Quo(fbal, big.NewFloat(math.Pow10(int(18))))
	log.Printf("%s bnb: %.18f\n", ea.Addr.Hex(), value)
	return balance, nil
}

func (ea *ExtAcc) GetBEP20Balance(contAddr *common.Address) (*big.Int, error) {
	balance, err := contracts.Bep20InstanceMap[contAddr.String()].BalanceOf(&bind.CallOpts{}, *ea.Addr)
	if err != nil {
		return nil, err
	}
	symbol, err := contracts.Bep20InstanceMap[contAddr.String()].Symbol(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	decimals, err := contracts.Bep20InstanceMap[contAddr.String()].Decimals(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	fbal := new(big.Float)
	fbal.SetString(balance.String())
	value := new(big.Float).Quo(fbal, big.NewFloat(math.Pow10(int(decimals))))
	log.Printf("%s %s-%s: %.18f\n", ea.Addr.Hex(), contAddr.Hex(), symbol, value)
	return balance, nil
}

func (ea *ExtAcc) GetReceipt(hash *common.Hash, waitTimeSec int64) *types.Receipt {
	for {
		ctx := context.Background()
		receipt, err := ea.Client.TransactionReceipt(ctx, *hash)
		if err != nil {
			if err.Error() == "not found" {
				log.Printf("%s is in process\n", hash.Hex())
			} else {
				log.Println("error:", err.Error())
				return nil
			}
		}
		if receipt != nil {
			log.Printf("%s status: %d, gasUsed: %d\n",
				hash.Hex(), receipt.Status, receipt.GasUsed)
			return receipt
		}
		time.Sleep(time.Duration(waitTimeSec) * time.Second)
	}
}

func (ea *ExtAcc) GetBlockTrans(start int64, end int64) {
	ctx := context.Background()
	var txns uint64
	var gasPerTx uint64
	var allGasUsed uint64
	for i := start; i < end; i++ {
		blockNum := big.NewInt(i)
		block, err := ea.Client.BlockByNumber(ctx, blockNum)
		if err != nil {
			if err.Error() == "not found" {
				break
			}
			log.Println("error:", err.Error())
			continue
		}
		count := uint64(len(block.Transactions()))
		var gasPerTx uint64
		if count == 0 {
			gasPerTx = 0
		} else {
			gasPerTx = block.GasUsed() / count
		}
		log.Printf("#%d, %s, D: %d, %v, TX: %d, L: %d, U: %d, %d\n",
			block.Number().Uint64(),
			block.Coinbase().Hex(),
			block.Difficulty().Uint64(),
			time.Unix(int64(block.Time()), 0),
			len(block.Transactions()),
			block.GasLimit(),
			block.GasUsed(),
			gasPerTx)
		txns += uint64(len(block.Transactions())) - 1
		allGasUsed += block.GasUsed()
	}
	gasPerTx = allGasUsed / txns
	log.Printf("from: %d to: %d txns: %d gasPerTx: %d", start, end, txns, gasPerTx)
}

func (ea *ExtAcc) BuildTransactOpts(nonce *uint64, gasLimit *uint64) (*bind.TransactOpts, error) {
	gasTipCap, err := ea.Client.SuggestGasTipCap(context.Background())
	if err != nil {
		return nil, err
	}

	transactOpts, err := bind.NewKeyedTransactorWithChainID(ea.Key, T_cfg.ChainId)
	if err != nil {
		return nil, err
	}
	transactOpts.Nonce = big.NewInt(int64(*nonce))
	transactOpts.Value = big.NewInt(0)
	transactOpts.GasLimit = *gasLimit
	transactOpts.GasFeeCap = gasTipCap.Add(gasTipCap, gasTipCap)
	transactOpts.GasTipCap = gasTipCap
	//
	return transactOpts, nil
}

func (ea *ExtAcc) BuildTransactOptsNoEip1559(nonce *uint64, gasLimit *uint64) (*bind.TransactOpts, error) {
	gasPrice := big.NewInt(6e10)
	gasTipCap, err := ea.Client.SuggestGasTipCap(context.Background())
	log.Println("gasTipCap", gasTipCap)
	if err != nil {
		return nil, err
	}
	//
	transactOpts, err := bind.NewKeyedTransactorWithChainID(ea.Key, T_cfg.ChainId)
	if err != nil {
		return nil, err
	}
	transactOpts.Nonce = big.NewInt(int64(*nonce))
	transactOpts.Value = big.NewInt(0)
	transactOpts.GasLimit = *gasLimit
	transactOpts.GasPrice = gasPrice
	//
	return transactOpts, nil
}

// native
func (ea *ExtAcc) SendBNB(nonce uint64, toAddr *common.Address, amount *big.Int) (*common.Hash, error) {
	gasLimit := uint64(21000)
	gasTipCap, err := ea.Client.SuggestGasTipCap(context.Background())
	if err != nil {
		return nil, err
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   T_cfg.ChainId,
		Nonce:     nonce,
		GasFeeCap: gasTipCap.Mul(gasTipCap, big.NewInt(2)),
		GasTipCap: gasTipCap,
		Gas:       gasLimit,
		To:        toAddr,
		Value:     amount,
	})
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(T_cfg.ChainId), ea.Key)
	if err != nil {
		return nil, err
	}
	err = ea.Client.SendTransaction(context.Background(), signedTx)
	if err != nil {

		log.Printf("SendTransaction: %v , to: %s \n", err, toAddr.Hex())
		return nil, err
	}
	hash := signedTx.Hash()
	log.Printf("amount %d send bnb: %s \n", amount.Int64(), hash.Hex())
	return &hash, nil
}

// native
func (ea *ExtAcc) SendBNBWithoutEIP1559(nonce uint64, toAddr *common.Address, amount *big.Int) (*common.Hash, error) {
	//
	gasLimit := uint64(23000)
	ctx := context.Background()
	gasPrice, err := ea.Client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	tx := types.NewTransaction(nonce, *toAddr, amount, gasLimit, gasPrice, nil)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(T_cfg.ChainId), ea.Key)
	if err != nil {
		return nil, err
	}
	err = ea.Client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, err
	}
	hash := signedTx.Hash()
	log.Printf("send bnb: %s\n", hash.Hex())
	return &hash, nil
}

// bep20
func (ea *ExtAcc) SendBEP20(nonce uint64, contAddr *common.Address, toAddr *common.Address, amount *big.Int) (*common.Hash, error) {
	//
	gasLimit := uint64(3e5)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}

	tx, err := contracts.Bep20InstanceMap[contAddr.String()].Transfer(transactOpts, *toAddr, amount)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("amount %d send bep20: %s \n", amount.Int64(), hash.Hex())
	return &hash, nil
}

// bep20
func (ea *ExtAcc) ApproveBEP20(nonce uint64, contAddr *common.Address, spenderAddr *common.Address, amount *big.Int) (*common.Hash, error) {
	gasLimit := uint64(3e5)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}

	tx, err := contracts.Bep20InstanceMap[contAddr.String()].Approve(transactOpts, *spenderAddr, amount)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("---- approve bep20: %s\n", hash.Hex())
	return &hash, nil
}

// uniswap
func (ea *ExtAcc) AddLiquidity(nonce uint64, token1Addr *common.Address, token2Addr *common.Address, amountADesired *big.Int, amountBDesired *big.Int, toAddr *common.Address) (*common.Hash, error) {
	gasLimit := uint64(5e6)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}

	deadline := big.NewInt(time.Now().Unix() + 300) // 100 blocks
	tx, err := contracts.V2RouterInstance.AddLiquidity(transactOpts, *token1Addr, *token2Addr, amountADesired, amountBDesired, big.NewInt(10000), big.NewInt(10000), *toAddr, deadline)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("gas %d tip %d add liquidity: %s ea: %s token1: %s token2: %s\n", transactOpts.GasLimit, transactOpts.GasTipCap, hash.Hex(), ea.Addr, *token1Addr, *token2Addr)
	return &hash, nil
}

// uniswap
func (ea *ExtAcc) GetPair(token1Addr *common.Address, token2Addr *common.Address) (*common.Address, error) {
	addr, err := contracts.FactoryInstance.GetPair(&bind.CallOpts{}, *token1Addr, *token2Addr)
	if err != nil {
		return nil, err
	}
	log.Printf("get pair: %s\n", addr.Hex())
	return &addr, nil
}

// uniswap
func (ea *ExtAcc) RemoveLiquidity(nonce uint64, token1Addr *common.Address, token2Addr *common.Address, liquidity *big.Int, toAddr *common.Address) (*common.Hash, error) {
	gasLimit := uint64(5e6)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}
	//
	deadline := big.NewInt(time.Now().Unix() + 300) // 100 blocks
	tx, err := contracts.V2RouterInstance.RemoveLiquidity(transactOpts, *token1Addr, *token2Addr, liquidity, big.NewInt(10000), big.NewInt(10000), *toAddr, deadline)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("remove liquidity: %s\n", hash.Hex())
	return &hash, nil
}

// uniswap, swap bep20 to bep20
func (ea *ExtAcc) SwapExactTokensForTokens(nonce uint64, amountIn *big.Int, path []common.Address, toAddr *common.Address) (*common.Hash, error) {
	gasLimit := uint64(5e6)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}

	deadline := big.NewInt(time.Now().Unix() + 300) // 100 blocks
	// todo: amountOutMin is set to 0 to tolerant possible slippage
	tx, err := contracts.V2RouterInstance.SwapExactTokensForTokens(transactOpts, amountIn, big.NewInt(0), path, *toAddr, deadline)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("swap exact tokens for tokens: %s\n", hash.Hex())
	return &hash, nil
}

// SwapBNBForExactTokens  swap bnb to bep20 (unused bnb will be returned to the caller)
func (ea *ExtAcc) SwapBNBForExactTokens(nonce uint64, amountIn *big.Int, amountOut *big.Int, path []common.Address, toAddr *common.Address) (*common.Hash, error) {
	gasLimit := uint64(5e6)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}

	transactOpts.Value = amountIn
	deadline := big.NewInt(time.Now().Unix() + 300) // 100 blocks
	tx, err := contracts.V2RouterInstance.SwapETHForExactTokens(transactOpts, amountOut, path, *toAddr, deadline)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("swap bnb for exact tokens: %s\n", hash.Hex())
	return &hash, nil
}

// SwapExactTokensForBNB swap bep20   to bnb (unused bnb will be returned to the caller)
func (ea *ExtAcc) SwapExactTokensForBNB(nonce uint64, amountIn *big.Int, path []common.Address, toAddr *common.Address) (*common.Hash, error) {
	gasLimit := uint64(5e6)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}
	//
	//transactOpts.Value = amountIn
	deadline := big.NewInt(time.Now().Unix() + 300) // 100 blocks
	tx, err := contracts.V2RouterInstance.SwapExactTokensForETH(transactOpts, amountIn, big.NewInt(0), path, *toAddr, deadline)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("swap bnb for exact tokens: %s\n", hash.Hex())
	return &hash, nil
}

// wbnb
func (ea *ExtAcc) DepositWBNB(nonce uint64, amount *big.Int) (*common.Hash, error) {
	gasLimit := uint64(3e5)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}

	transactOpts.Value = amount
	tx, err := contracts.WbnbInstance.Deposit(transactOpts)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("gas %d tip %d deposit wbnb: %s\n", transactOpts.GasLimit, transactOpts.GasTipCap, hash.Hex())
	return &hash, nil
}

// wbnb
func (ea *ExtAcc) WithdrawWBNB(nonce uint64, amount *big.Int) (*common.Hash, error) {
	gasLimit := uint64(3e5)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}

	tx, err := contracts.WbnbInstance.Withdraw(transactOpts, amount)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("withdraw wbnb: %s\n", hash.Hex())
	return &hash, nil
}

func (ea *ExtAcc) SimulateContCall(contAddr *common.Address, transactOpts *bind.TransactOpts, trans *types.Transaction) ([]byte, error) {
	msg := ethereum.CallMsg{From: *ea.Addr, To: contAddr, Gas: transactOpts.GasLimit, GasPrice: transactOpts.GasPrice,
		GasFeeCap: transactOpts.GasFeeCap, GasTipCap: transactOpts.GasTipCap, Value: transactOpts.Value,
		Data: trans.Data()}
	ctx := context.Background()
	res, err := ea.Client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// erc721
func (ea *ExtAcc) MintERC721(nonce uint64) (*common.Hash, error) {
	log.Println("erc721 mint action")
	gasLimit := uint64(8e6)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}

	tx, err := contracts.Erc721Instance.SafeMint(transactOpts, *ea.Addr)
	if err != nil {
		return nil, err
	}
	txHash := tx.Hash()
	log.Printf("mint NFT721: %s\n", txHash.Hex())
	return &txHash, nil
}

// erc721
func (ea *ExtAcc) ApproveERC721(nonce uint64, spenderAddr *common.Address, amount *big.Int) (*common.Hash, error) {
	gasLimit := uint64(3e5)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}

	tx, err := contracts.Erc721Instance.Approve(transactOpts, *spenderAddr, amount)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("Approve NFT721: %s\n", hash.Hex())
	return &hash, nil
}

// erc721
func (ea *ExtAcc) TransferERC721(nonce uint64, toAddr *common.Address, TokenID *big.Int) (*common.Hash, error) {
	log.Println("erc721 transfer action")
	gasLimit := uint64(3e5)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}
	tx, err := contracts.Erc721Instance.SafeTransferFrom(transactOpts, *ea.Addr, *toAddr, TokenID)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("transfer nft: %s\n", hash.Hex())
	return &hash, nil
}

// erc721
func (ea *ExtAcc) Get721TotalSupply() (*big.Int, error) {
	totalSupply, err := contracts.Erc721Instance.TotalSupply(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	return totalSupply, nil
}

// erc721
func (ea *ExtAcc) GetOneERC721TokenID() (*big.Int, error) {
	tokenID, err := contracts.Erc721Instance.TokenOfOwnerByIndex(&bind.CallOpts{}, *ea.Addr, big.NewInt(0))
	if err != nil {
		return nil, err
	}
	return tokenID, nil
}

// erc1155
func (ea *ExtAcc) MintERC1155(nonce uint64, tokenID, amount *big.Int) (*common.Hash, error) {
	log.Println("erc1155 mint action")
	gasLimit := uint64(8e6)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}
	tx, err := contracts.Erc1155Instance.Mint(transactOpts, *ea.Addr, tokenID, amount, []byte{0x00})
	if err != nil {
		return nil, err
	}
	txHash := tx.Hash()
	log.Printf("Mint NFT1155: %s\n", txHash.Hex())
	return &txHash, nil
}

func (ea *ExtAcc) MintBatchERC1155(nonce uint64, tokenID, amount []*big.Int) (*common.Hash, error) {
	gasLimit := uint64(8e6)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}
	tx, err := contracts.Erc1155Instance.MintBatch(transactOpts, *ea.Addr, tokenID, amount, []byte{0x00})
	if err != nil {
		return nil, err
	}
	txHash := tx.Hash()
	log.Printf("MintBatch NFT1155: %s\n", txHash.Hex())
	return &txHash, nil
}

func (ea *ExtAcc) BurnERC1155(nonce uint64, tokenID, amount *big.Int) (*common.Hash, error) {
	log.Println("erc1155 burn action")
	gasLimit := uint64(3e5)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}

	tx, err := contracts.Erc1155Instance.Burn(transactOpts, *ea.Addr, tokenID, amount)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("Burn NFT1155: %s\n", hash.Hex())
	return &hash, nil
}

func (ea *ExtAcc) GetOneERC1155TokenID(tokenIDSlice []*big.Int) (*big.Int, error) {
	var accounts []common.Address
	var isEmpty = true
	var tokenID *big.Int

	for range tokenIDSlice {
		accounts = append(accounts, *ea.Addr)
	}
	balance, err := contracts.Erc1155Instance.BalanceOfBatch(&bind.CallOpts{}, accounts, tokenIDSlice)
	if err != nil {
		return nil, err
	}
	for i := range balance {
		if balance[i].Cmp(big.NewInt(0)) != 0 {
			isEmpty = false
			tokenID = tokenIDSlice[i]
			break
		}
	}
	if isEmpty {
		err = fmt.Errorf("balance is 0")
		return nil, err
	}
	return tokenID, nil
}

// erc1155
func (ea *ExtAcc) TransERC1155(nonce uint64, toAddr common.Address, tokenID, amount *big.Int) (*common.Hash, error) {
	log.Println("erc1155 trans action")
	gasLimit := uint64(6e5)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}

	tx, err := contracts.Erc1155Instance.SafeTransferFrom(transactOpts, *ea.Addr, toAddr, tokenID, amount, []byte{0x00})
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("Trans NFT1155: %s\n", hash.Hex())
	return &hash, nil
}
