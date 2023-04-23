package executor

import (
	"bsc-load-test/utils"
	"context"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/ratelimit"
)

// actully, the real tps when runing initTestAcc command is nearly 3*Tps ~ 8*Tps
func InitAccount(clients []*ethclient.Client, nonce uint64, root utils.ExtAcc) {
	eaSlice := utils.Load(clients, utils.T_cfg.Hexkeyfile, &utils.T_cfg.UsersLoaded)
	limiter := ratelimit.New(utils.T_cfg.Tps)
	//
	slaveEaSlice := utils.Load(clients, utils.T_cfg.SlaveUserHexkeyFile, &utils.T_cfg.SlaveUserLoaded)
	startTime := time.Now()
	InitCoins(limiter, nonce, root, eaSlice, slaveEaSlice)
	endTime := time.Now()
	log.Printf("init account: send coins total %f seconds.", endTime.Sub(startTime).Seconds())
	InitSingleAccount(limiter, nonce, root, eaSlice, slaveEaSlice)
	endTime = time.Now()
	log.Printf("init account: finished, total %f seconds.", endTime.Sub(startTime).Seconds())

}

func InitCoins(limiter ratelimit.Limiter, nonce uint64, root utils.ExtAcc, eaSlice, slaveEaSlice []utils.ExtAcc) {
	// send coin to root accounts
	log.Println("init account: send coins to slave accounts.")
	for i, v := range slaveEaSlice {
		limiter.Take()
		//
		_, err := root.SendBNB(nonce, v.Addr, utils.T_cfg.SlaveDistributeAmount)
		if err != nil {
			log.Println("error: send bnb:", err)
			continue
		}
		nonce++
		//
		if utils.T_cfg.Bep20Hex != "" {
			//
			index := i % len(utils.T_cfg.Bep20AddrsA)
			//
			_, err = root.SendBEP20(nonce, &utils.T_cfg.Bep20AddrsA[index], v.Addr, utils.T_cfg.SlaveDistributeAmount)
			if err != nil {
				log.Println("error: send bep20:", err)
				continue
			}
			nonce++
			//
			_, err = root.SendBEP20(nonce, &utils.T_cfg.Bep20AddrsB[index], v.Addr, utils.T_cfg.SlaveDistributeAmount)
			if err != nil {
				log.Println("error: send bep20:", err)
				continue
			}
			nonce++
		}
	}

	time.Sleep(5 * time.Second)

	// send coins to final accounts
	log.Println("init account: send coins to final accounts.")
	var slaveWg sync.WaitGroup
	slaveWg.Add(len(slaveEaSlice))
	for i, v := range slaveEaSlice {
		limiter.Take()
		go func(wg *sync.WaitGroup, i int, ea utils.ExtAcc) {
			defer wg.Done()
			capNumber := utils.T_cfg.UsersLoaded / utils.T_cfg.SlaveUserLoaded

			slaveNonce, err := ea.Client.PendingNonceAt(context.Background(), *ea.Addr)
			if err != nil {
				panic(err)
			}
			log.Printf("slave %d: nonce - %d \n", i, slaveNonce)

			for batchIndex, addr := range eaSlice[i*capNumber : (i+1)*capNumber] {
				limiter.Take()
				//
				_, err = ea.SendBNB(slaveNonce, addr.Addr, utils.T_cfg.DistributeAmount)
				if err != nil {
					log.Printf("slave %d child %d amount %d error: send bnb: %s \n", i, batchIndex, utils.T_cfg.DistributeAmount.Int64(), err)
					continue
				}
				slaveNonce++
				//
				if utils.T_cfg.Bep20Hex != "" {
					index := i % len(utils.T_cfg.Bep20AddrsA)
					//
					_, err = ea.SendBEP20(slaveNonce, &utils.T_cfg.Bep20AddrsA[index], addr.Addr, utils.T_cfg.DistributeAmount)
					if err != nil {
						log.Printf("slave %d child %d amount %d error: send bep20: %s \n", i, batchIndex, utils.T_cfg.DistributeAmount.Int64(), err)
						continue
					}

					slaveNonce++
					//
					_, err = ea.SendBEP20(slaveNonce, &utils.T_cfg.Bep20AddrsB[index], addr.Addr, utils.T_cfg.DistributeAmount)
					if err != nil {
						log.Printf("slave %d child %d amount %d error: send bep20: %s \n", i, batchIndex, utils.T_cfg.DistributeAmount.Int64(), err)
						continue
					}
					slaveNonce++
				}
			}
		}(&slaveWg, i, v)
	}
	slaveWg.Wait()
}

func InitSingleAccount(limiter ratelimit.Limiter, nonce uint64, root utils.ExtAcc, eaSlice, slaveEaSlice []utils.ExtAcc) {
	var tokenAmountSlice []*big.Int
	for range utils.T_cfg.Erc1155TokenIDSlice {
		tokenAmountSlice = append(tokenAmountSlice, big.NewInt(int64(utils.T_cfg.Erc1155InitTokenNumber)))
	}

	var wg sync.WaitGroup
	wg.Add(len(eaSlice))
	for i, v := range eaSlice {
		limiter.Take()
		go func(wg *sync.WaitGroup, i int, ea utils.ExtAcc) {
			defer wg.Done()
			//
			capNumber := utils.T_cfg.UsersLoaded / utils.T_cfg.SlaveUserLoaded
			//
			index := (i / capNumber) % len(utils.T_cfg.Bep20AddrsA)
			err := initUniswapAndNftByAcc(&ea, &utils.T_cfg.Bep20AddrsA[index], &utils.T_cfg.Bep20AddrsB[index], tokenAmountSlice)

			if err != nil {
				log.Println("error: initUniswapByAcc:", err)
				return
			}
		}(&wg, i, v)
	}
	wg.Wait()
}

func initUniswapAndNftByAcc(acc *utils.ExtAcc, tokenA *common.Address, tokenB *common.Address, tokenAmountSlice []*big.Int) error {
	nonce, err := acc.Client.PendingNonceAt(context.Background(), *acc.Addr)
	if err != nil {
		log.Println("error: nonce:", err)
		return err
	}
	wbnbAmount := new(big.Int)
	// doubled, one for balance; the other for add liquidity
	wbnbAmount.Mul(utils.T_cfg.LiquidityInitAmount, big.NewInt(2))
	_, err = acc.DepositWBNB(nonce, wbnbAmount)
	if err != nil {
		log.Println("error: deposit wbnb: " + err.Error())
		return err
	}
	nonce++
	_, err = acc.ApproveBEP20(nonce, &utils.T_cfg.WbnbAddr, &utils.T_cfg.UniswapRouterAddr, utils.T_cfg.DistributeAmount)
	if err != nil {
		log.Println("error: approve wbnb: " + err.Error())
		return err
	}
	nonce++
	_, err = acc.ApproveBEP20(nonce, tokenA, &utils.T_cfg.UniswapRouterAddr, utils.T_cfg.DistributeAmount)
	if err != nil {
		log.Println("error: approve bep20: " + err.Error())
		return err
	}
	nonce++
	_, err = acc.ApproveBEP20(nonce, tokenB, &utils.T_cfg.UniswapRouterAddr, utils.T_cfg.DistributeAmount)
	if err != nil {
		log.Println("error: approve bep20: " + err.Error())
		return err
	}
	nonce++
	_, err = acc.AddLiquidity(nonce, &utils.T_cfg.WbnbAddr, tokenA, utils.T_cfg.LiquidityInitAmount, utils.T_cfg.LiquidityInitAmount, acc.Addr)
	if err != nil {
		log.Println("error: add liquidity: " + err.Error())
		return err
	}
	nonce++
	_, err = acc.AddLiquidity(nonce, tokenA, tokenB, utils.T_cfg.LiquidityInitAmount, utils.T_cfg.LiquidityInitAmount, acc.Addr)
	if err != nil {
		log.Println("error: add liquidity: " + err.Error())
		return err
	}
	nonce++
	for i := 0; int64(i) < utils.T_cfg.Erc721InitTokenNumber; i++ {
		_, err = acc.MintERC721(nonce)
		if err != nil {
			log.Println("error: mint erc721:", err)
			return err
		}
		nonce++
	}
	_, err = acc.MintBatchERC1155(nonce, utils.T_cfg.Erc1155TokenIDSlice, tokenAmountSlice)
	if err != nil {
		log.Println("error: mint batch erc1155:", err)
		return err
	}
	return nil
}
