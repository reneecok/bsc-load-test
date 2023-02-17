package utils

import (
	"context"
	"crypto/ecdsa"
	"log"
	"math"
	"math/big"
	"time"

	"bsc-load-test/contracts/V2factory"
	"bsc-load-test/contracts/V2router"
	"bsc-load-test/contracts/bep20"
	"bsc-load-test/contracts/wbnb"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

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
	return &ExtAcc{client, key, &addr}, nil
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
	instance, err := bep20.NewBep20(*contAddr, ea.Client)
	if err != nil {
		return nil, err
	}
	balance, err := instance.BalanceOf(&bind.CallOpts{}, *ea.Addr)
	if err != nil {
		return nil, err
	}
	symbol, err := instance.Symbol(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	decimals, err := instance.Decimals(&bind.CallOpts{})
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
	}
}

func (ea *ExtAcc) BuildTransactOpts(nonce *uint64, gasLimit *uint64) (*bind.TransactOpts, error) {
	gasPrice, err := ea.Client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}

	chainID, err := ea.Client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}
	//
	transactOpts, err := bind.NewKeyedTransactorWithChainID(ea.Key, chainID)
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
	//
	gasLimit := uint64(23000)
	ctx := context.Background()
	gasPrice, err := ea.Client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	tx := types.NewTransaction(nonce, *toAddr, amount, gasLimit, gasPrice, nil)
	chainId, err := ea.Client.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainId), ea.Key)
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
	//
	instance, err := bep20.NewBep20(*contAddr, ea.Client)
	if err != nil {
		return nil, err
	}
	tx, err := instance.Transfer(transactOpts, *toAddr, amount)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("send bep20: %s\n", hash.Hex())
	return &hash, nil
}

// bep20
func (ea *ExtAcc) ApproveBEP20(nonce uint64, contAddr *common.Address, spenderAddr *common.Address, amount *big.Int) (*common.Hash, error) {
	gasLimit := uint64(3e5)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}
	//
	instance, err := bep20.NewBep20(*contAddr, ea.Client)
	if err != nil {
		return nil, err
	}
	tx, err := instance.Approve(transactOpts, *spenderAddr, amount)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("---- approve bep20: %s\n", hash.Hex())
	return &hash, nil
}

// uniswap
func (ea *ExtAcc) AddLiquidity(nonce uint64, contAddr *common.Address, token1Addr *common.Address, token2Addr *common.Address, amountADesired *big.Int, amountBDesired *big.Int, toAddr *common.Address) (*common.Hash, error) {
	gasLimit := uint64(5e6)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}
	//
	instance, err := V2router.NewV2router(*contAddr, ea.Client)
	if err != nil {
		return nil, err
	}
	deadline := big.NewInt(time.Now().Unix() + 300) // 100 blocks
	tx, err := instance.AddLiquidity(transactOpts, *token1Addr, *token2Addr, amountADesired, amountBDesired, big.NewInt(10000), big.NewInt(10000), *toAddr, deadline)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("add liquidity: %s\n", hash.Hex())
	return &hash, nil
}

// uniswap
func (ea *ExtAcc) GetPair(contAddr *common.Address, token1Addr *common.Address, token2Addr *common.Address) (*common.Address, error) {
	instance, err := V2factory.NewV2factory(*contAddr, ea.Client)
	if err != nil {
		return nil, err
	}
	addr, err := instance.GetPair(&bind.CallOpts{}, *token1Addr, *token2Addr)
	if err != nil {
		return nil, err
	}
	log.Printf("get pair: %s\n", addr.Hex())
	return &addr, nil
}

// uniswap
func (ea *ExtAcc) RemoveLiquidity(nonce uint64, contAddr *common.Address, token1Addr *common.Address, token2Addr *common.Address, liquidity *big.Int, toAddr *common.Address) (*common.Hash, error) {
	gasLimit := uint64(5e6)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}
	//
	instance, err := V2router.NewV2router(*contAddr, ea.Client)
	if err != nil {
		return nil, err
	}
	deadline := big.NewInt(time.Now().Unix() + 300) // 100 blocks
	tx, err := instance.RemoveLiquidity(transactOpts, *token1Addr, *token2Addr, liquidity, big.NewInt(10000), big.NewInt(10000), *toAddr, deadline)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("remove liquidity: %s\n", hash.Hex())
	return &hash, nil
}

// uniswap, swap bep20 to bep20
func (ea *ExtAcc) SwapExactTokensForTokens(nonce uint64, contAddr *common.Address, amountIn *big.Int, path []common.Address, toAddr *common.Address) (*common.Hash, error) {
	gasLimit := uint64(5e6)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}
	//
	instance, err := V2router.NewV2router(*contAddr, ea.Client)
	if err != nil {
		return nil, err
	}
	deadline := big.NewInt(time.Now().Unix() + 300) // 100 blocks
	// todo: amountOutMin is set to 0 to tolerant possible slippage
	tx, err := instance.SwapExactTokensForTokens(transactOpts, amountIn, big.NewInt(0), path, *toAddr, deadline)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("swap exact tokens for tokens: %s\n", hash.Hex())
	return &hash, nil
}

// uniswap, swap bnb to bep20 (unused bnb will be returned to the caller)
func (ea *ExtAcc) SwapBNBForExactTokens(nonce uint64, contAddr *common.Address, amountIn *big.Int, amountOut *big.Int, path []common.Address, toAddr *common.Address) (*common.Hash, error) {
	gasLimit := uint64(5e6)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}
	//
	instance, err := V2router.NewV2router(*contAddr, ea.Client)
	if err != nil {
		return nil, err
	}
	transactOpts.Value = amountIn
	deadline := big.NewInt(time.Now().Unix() + 300) // 100 blocks
	tx, err := instance.SwapETHForExactTokens(transactOpts, amountOut, path, *toAddr, deadline)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("swap bnb for exact tokens: %s\n", hash.Hex())
	return &hash, nil
}

// wbnb
func (ea *ExtAcc) DepositWBNB(nonce uint64, contAddr *common.Address, amount *big.Int) (*common.Hash, error) {
	gasLimit := uint64(3e5)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}
	//
	instance, err := wbnb.NewWbnb(*contAddr, ea.Client)
	if err != nil {
		return nil, err
	}
	transactOpts.Value = amount
	tx, err := instance.Deposit(transactOpts)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("deposit wbnb: %s\n", hash.Hex())
	return &hash, nil
}

// wbnb
func (ea *ExtAcc) WithdrawWBNB(nonce uint64, contAddr *common.Address, amount *big.Int) (*common.Hash, error) {
	gasLimit := uint64(3e5)
	transactOpts, err := ea.BuildTransactOpts(&nonce, &gasLimit)
	if err != nil {
		return nil, err
	}
	//
	instance, err := wbnb.NewWbnb(*contAddr, ea.Client)
	if err != nil {
		return nil, err
	}
	tx, err := instance.Withdraw(transactOpts, amount)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()
	log.Printf("withdraw wbnb: %s\n", hash.Hex())
	return &hash, nil
}

func (ea *ExtAcc) SimulateContCall(contAddr *common.Address, transactOpts *bind.TransactOpts, trans *types.Transaction) ([]byte, error) {
	msg := ethereum.CallMsg{*ea.Addr, contAddr, transactOpts.GasLimit, transactOpts.GasPrice, transactOpts.Value, trans.Data(), nil}
	ctx := context.Background()
	res, err := ea.Client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, err
	}
	return res, nil
}
