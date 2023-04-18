package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bsc-load-test/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/ratelimit"
)

var randTestAcc *bool
var initTestAcc *bool
var resetTestAcc *bool
var runTestAcc *bool

var tps *int
var sec *int

var queryBlocks *bool
var blockNumS *int64
var blockNumE *int64

var debug *bool

func init() {
	rand.Seed(time.Now().UnixNano())

	randTestAcc = flag.Bool("randTestAcc", false, "randTestAcc")
	initTestAcc = flag.Bool("initTestAcc", false, "initTestAcc")
	resetTestAcc = flag.Bool("resetTestAcc", false, "resetTestAcc")
	runTestAcc = flag.Bool("runTestAcc", false, "runTestAcc")
	queryBlocks = flag.Bool("queryBlocks", false, "queryBlocks")

	tps = flag.Int("tps", -10, "tps")
	sec = flag.Int("sec", -10, "sec")

	blockNumS = flag.Int64("blockNumS", 0, "blockNumS")
	blockNumE = flag.Int64("blockNumE", 0, "blockNumE")
	debug = flag.Bool("debug", false, "debug")

	flag.Parse()

	// init config from config.yml
	err := utils.T_cfg.LoadYml(tps, sec)
	if err != nil {
		log.Panicln(err)
	}
}

func main() {
	//
	if *randTestAcc {
		utils.RandHexKeys(utils.T_cfg.Hexkeyfile, utils.T_cfg.UsersCreated)
		return
	}
	//
	clients := make([]*ethclient.Client, 0, len(utils.T_cfg.Fullnodes))
	for i, v := range utils.T_cfg.Fullnodes {
		log.Printf("%d: %s", i, v)
		client, err := ethclient.Dial(v)
		if err != nil {
			panic(err)
		}
		clients = append(clients, client)
	}
	defer cleanup(clients)
	//
	root, err := utils.NewExtAcc(clients[0], utils.T_cfg.Roothexkey, utils.T_cfg.Roothexaddr)
	if err != nil {
		panic(err)
	}
	log.Println("root:", root.Addr.Hex())
	//
	_, err = root.GetBNBBalance()
	if err != nil {
		panic(err)
	}
	//
	if utils.T_cfg.Bep20Hex != "" {
		for i := range utils.T_cfg.Bep20AddrsA {
			_, err = root.GetBEP20Balance(&utils.T_cfg.Bep20AddrsA[i])
			if err != nil {
				panic(err)
			}
			_, err = root.GetBEP20Balance(&utils.T_cfg.Bep20AddrsB[i])
			if err != nil {
				panic(err)
			}
		}
	}
	if utils.T_cfg.WbnbHex != "" {
		_, err = root.GetBEP20Balance(&utils.T_cfg.WbnbAddr)
		if err != nil {
			panic(err)
		}
	}
	//
	nonce, err := root.Client.PendingNonceAt(context.Background(), *root.Addr)
	if err != nil {
		panic(err)
	}
	log.Println("root: nonce -", nonce)
	//
	if *initTestAcc {
		startTime := time.Now()
		//
		limiter := ratelimit.New(*tps)
		//
		eaSlice := load(clients, utils.T_cfg.Hexkeyfile, &utils.T_cfg.UsersLoaded)
		slaveEaSlice := load(clients, utils.T_cfg.SlaveUserHexkeyFile, &utils.T_cfg.SlaveUserLoaded)
		time.Sleep(10 * time.Second)

		//send coin to root accounts
		for i, v := range slaveEaSlice {
			limiter.Take()
			//
			_, err = root.SendBNB(nonce, v.Addr, utils.T_cfg.SlaveDistributeAmount)
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
		time.Sleep(10 * time.Second)

		// send coin to final accounts
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
		endTime := time.Now()
		times := endTime.Sub(startTime).Seconds()
		log.Printf("init_before %f seconds \n", times)

		time.Sleep(10 * time.Second)
		//
		if utils.T_cfg.WbnbHex != "" && utils.T_cfg.UniswapFactoryHex != "" && utils.T_cfg.UniswapRouterHex != "" {
			//
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
					err = initUniswapByAcc(&ea, &utils.T_cfg.Bep20AddrsA[index], &utils.T_cfg.Bep20AddrsB[index])
					if err != nil {
						log.Println("error: initUniswapByAcc:", err)
						return
					}
				}(&wg, i, v)
			}
			wg.Wait()
		}

		if utils.T_cfg.Erc721Hex != "" {
			var wg sync.WaitGroup
			var totalAccountSlice []utils.ExtAcc
			for i := 0; i < int(utils.T_cfg.Erc721InitTokenNumber); i++ {
				totalAccountSlice = append(totalAccountSlice, eaSlice...)
			}
			wg.Add(len(totalAccountSlice))
			for _, v := range totalAccountSlice {
				limiter.Take()
				go func(wg *sync.WaitGroup, v utils.ExtAcc) {
					defer wg.Done()
					nonce, err = v.Client.PendingNonceAt(context.Background(), *v.Addr)
					if err != nil {
						log.Println("error: get nonce in mint erc721:", err)
						return
					}
					_, err = v.MintERC721(nonce, utils.T_cfg.Erc721Addr)
					if err != nil {
						log.Println("error: mint erc721:", err)
						return
					}
				}(&wg, v)
			}
			wg.Wait()
		}

		if utils.T_cfg.Erc1155Hex != "" {
			var wg sync.WaitGroup
			wg.Add(len(eaSlice))
			var tokenAmountSlice []*big.Int
			for range utils.T_cfg.Erc1155TokenIDSlice {
				tokenAmountSlice = append(tokenAmountSlice, big.NewInt(int64(utils.T_cfg.Erc1155InitTokenNumber)))
			}
			for _, v := range eaSlice {
				limiter.Take()
				go func(wg *sync.WaitGroup, v utils.ExtAcc) {
					defer wg.Done()
					nonce, err = v.Client.PendingNonceAt(context.Background(), *v.Addr)
					if err != nil {
						log.Println("error: get nonce in mint erc1155:", err)
						return
					}
					_, err = v.MintBatchERC1155(nonce, utils.T_cfg.Erc1155Addr, utils.T_cfg.Erc1155TokenIDSlice, tokenAmountSlice)
					if err != nil {
						log.Println("error: mint batch erc1155:", err)
						return
					}
				}(&wg, v)
			}
			wg.Wait()
		}

		endTime = time.Now()
		times = endTime.Sub(startTime).Seconds()
		log.Printf("init_acc_time %f seconds \n", times)
		return
	}
	//
	if *resetTestAcc {
		//
		limiter := ratelimit.New(*tps)
		//
		var wg sync.WaitGroup
		eaSlice := load(clients, utils.T_cfg.Hexkeyfile, &utils.T_cfg.UsersLoaded)
		//
		if utils.T_cfg.WbnbHex != "" {
			wg.Add(len(eaSlice))
			for _, ea := range eaSlice {
				limiter.Take()
				go func(wg *sync.WaitGroup, ea utils.ExtAcc) {
					defer wg.Done()
					//
					nonce, err := ea.Client.PendingNonceAt(
						context.Background(), *ea.Addr)
					if err != nil {
						log.Println("error: nonce:", err)
						return
					}
					//
					balance, err := ea.GetBEP20Balance(&utils.T_cfg.WbnbAddr)
					if err != nil {
						log.Println("error: get wbnb balance: %s, %s\n",
							ea.Addr.Hex(), err)
						return
					}
					_, err = ea.WithdrawWBNB(nonce, &utils.T_cfg.WbnbAddr, balance)
					if err != nil {
						log.Println("error: withdraw wbnb:", err)
						return
					}
				}(&wg, ea)
			}
			wg.Wait()
		}
		//
		time.Sleep(3 * time.Second)
		//
		wg.Add(len(eaSlice))
		for _, ea := range eaSlice {
			limiter.Take()
			go func(wg *sync.WaitGroup, ea utils.ExtAcc) {
				defer wg.Done()
				//
				nonce, err := ea.Client.PendingNonceAt(context.Background(), *ea.Addr)
				if err != nil {
					log.Println("error: nonce:", err)
					return
				}
				//
				balance, err := ea.GetBNBBalance()
				if err != nil {
					log.Printf("error: get bnb balance: %s, %s\n",
						ea.Addr.Hex(), err)
					return
				}
				base := big.NewInt(1e16)
				if balance.Cmp(base) <= 0 {
					return
				} else {
					balance.Sub(balance, base)
				}
				_, err = ea.SendBNB(nonce, root.Addr, balance)
				if err != nil {
					log.Println("error: send bnb:", err)
					return
				}
			}(&wg, ea)
		}
		wg.Wait()
		//
		return
	}
	//
	if *runTestAcc {
		eaSlice := load(clients, utils.T_cfg.Hexkeyfile, &utils.T_cfg.UsersLoaded)
		block, err := root.Client.BlockByNumber(
			context.Background(), nil)
		if err != nil {
			panic(err)
		}
		log.Printf("the latest block: %d\n", block.Number().Uint64())
		results := exec(eaSlice)
		log.Println("# tx hash returned in load test:", len(results))
		// check all transaction status
		finishedNumber := checkAllTransactionStatus(root, results)
		log.Println("# tx finished in load test:", finishedNumber)
		dir := filepath.Dir(utils.T_cfg.Hexkeyfile)
		suffix := time.Now().UnixNano()
		fullpath := filepath.Join(dir, "results", fmt.Sprintf("results_%d.csv", suffix))
		utils.SaveHash(fullpath, results)
		return
	}
	//
	if *queryBlocks {
		root.GetBlockTrans(*blockNumS, *blockNumE)
		return
	}
}

func cleanup(clients []*ethclient.Client) {
	for _, v := range clients {
		v.Close()
	}
}

func load(clients []*ethclient.Client, hexkeyfile string, usersLoaded *int) []utils.ExtAcc {
	batches := utils.LoadHexKeys(hexkeyfile, *usersLoaded)
	eaSlice := make([]utils.ExtAcc, 0, *usersLoaded)
	//
	start := time.Now()
	var wg sync.WaitGroup
	var mx sync.Mutex
	for i, batch := range batches {
		wg.Add(1)
		go func(wg *sync.WaitGroup, i int, batch []string) {
			defer wg.Done()
			log.Printf("processing ea batch [%d]", i)
			for j, v := range batch {
				client := clients[j%len(clients)]
				items := strings.Split(v, ",")
				ea, err := utils.NewExtAcc(client, items[0], items[1])
				if err != nil {
					panic(err.Error())
				}
				mx.Lock()
				eaSlice = append(eaSlice, *ea)
				mx.Unlock()
			}
		}(&wg, i, batch)
	}
	wg.Wait()
	//
	end := time.Now()
	log.Printf("ea load time (ms): %d",
		end.Sub(start).Milliseconds())
	log.Printf("%d loaded", len(eaSlice))
	return eaSlice
}

func initUniswapByAcc(acc *utils.ExtAcc, tokenA *common.Address, tokenB *common.Address) error {
	nonce, err := acc.Client.PendingNonceAt(context.Background(), *acc.Addr)
	if err != nil {
		log.Println("error: nonce:", err)
		return err
	}
	wbnbAmount := new(big.Int)
	// doubled, one for balance; the other for add liquidity
	wbnbAmount.Mul(utils.T_cfg.LiquidityInitAmount, big.NewInt(2))
	_, err = acc.DepositWBNB(nonce, &utils.T_cfg.WbnbAddr, wbnbAmount)
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
	_, err = acc.AddLiquidity(nonce, &utils.T_cfg.UniswapRouterAddr, &utils.T_cfg.WbnbAddr, tokenA, utils.T_cfg.LiquidityInitAmount, utils.T_cfg.LiquidityInitAmount, acc.Addr)
	if err != nil {
		log.Println("error: add liquidity: " + err.Error())
		return err
	}
	nonce++
	_, err = acc.AddLiquidity(nonce, &utils.T_cfg.UniswapRouterAddr, tokenA, tokenB, utils.T_cfg.LiquidityInitAmount, utils.T_cfg.LiquidityInitAmount, acc.Addr)
	if err != nil {
		log.Println("error: add liquidity: " + err.Error())
		return err
	}
	return nil
}

func exec(eaSlice []utils.ExtAcc) []*common.Hash {
	//
	limiter := ratelimit.New(*tps)
	dur := time.Duration(*sec) * time.Second
	expired := setupTimer(dur)
	trans := (*tps) * (*sec)
	results := make([]*common.Hash, 0, trans)
	//
	var wg sync.WaitGroup
	var m sync.Mutex
	//
	i := 0
	for {
		limiter.Take()
		randomAddress := eaSlice[rand.Intn(len(eaSlice))]
		if *expired {
			break
		}
		wg.Add(1)
		go func(wg *sync.WaitGroup, i int, ea, randomAddress *utils.ExtAcc) {
			defer wg.Done()
			//
			scenario := utils.RandScenario(utils.T_cfg.Scenarios)
			//
			nonce, err := ea.Client.PendingNonceAt(context.Background(), *ea.Addr)
			if err != nil {
				log.Println("error: nonce:", err)
				return
			}
			//
			capNumber := utils.T_cfg.UsersLoaded / utils.T_cfg.SlaveUserLoaded
			//
			j := rand.Intn(utils.T_cfg.UsersLoaded)
			index := (i / capNumber) % len(utils.T_cfg.Bep20AddrsA)
			//
			var hash *common.Hash
			//
			switch scenario.Name {
			case utils.SendBNB:
				hash, err = ea.SendBNB(nonce, eaSlice[j].Addr, utils.T_cfg.LiquidityTestAmount)
				if err != nil {
					log.Println("error: send bnb:", err)
					return
				}
			case utils.SendBEP20:
				hash, err = ea.SendBEP20(nonce, &utils.T_cfg.Bep20AddrsA[index], eaSlice[j].Addr, utils.T_cfg.LiquidityTestAmount)
				if err != nil {
					log.Println("error: send bep20:", err)
					return
				}
			case utils.AddLiquidity:
				r := rand.Intn(10000) % 2
				if r == 0 {
					// bep20-bep20
					_, err = ea.ApproveBEP20(nonce, &utils.T_cfg.Bep20AddrsA[index], &utils.T_cfg.UniswapRouterAddr, utils.T_cfg.DistributeAmount)
					if err != nil {
						log.Println("error: approve bep20:", err)
						return
					}
					nonce++
					_, err = ea.ApproveBEP20(nonce, &utils.T_cfg.Bep20AddrsB[index], &utils.T_cfg.UniswapRouterAddr, utils.T_cfg.DistributeAmount)
					if err != nil {
						log.Println("error: approve bep20:", err)
						return
					}
					nonce++
					hash, err = ea.AddLiquidity(nonce, &utils.T_cfg.UniswapRouterAddr, &utils.T_cfg.Bep20AddrsA[index],
						&utils.T_cfg.Bep20AddrsB[index], utils.T_cfg.LiquidityTestAmount, utils.T_cfg.LiquidityTestAmount, ea.Addr)
					if err != nil {
						log.Println("error: add liquidity:", err)
						return
					}
				}
				if r == 1 {
					// wbnb-bep20
					_, err = ea.ApproveBEP20(nonce, &utils.T_cfg.WbnbAddr, &utils.T_cfg.UniswapRouterAddr, utils.T_cfg.DistributeAmount)
					if err != nil {
						log.Println("error: approve wbnb:", err)
						return
					}
					nonce++
					_, err = ea.ApproveBEP20(nonce, &utils.T_cfg.Bep20AddrsA[index], &utils.T_cfg.UniswapRouterAddr, utils.T_cfg.DistributeAmount)
					if err != nil {
						log.Println("error: approve bep20:", err)
						return
					}
					nonce++
					hash, err = ea.AddLiquidity(nonce, &utils.T_cfg.UniswapRouterAddr, &utils.T_cfg.WbnbAddr, &utils.T_cfg.Bep20AddrsA[index],
						utils.T_cfg.LiquidityTestAmount, utils.T_cfg.LiquidityTestAmount, ea.Addr)
					if err != nil {
						log.Println("error: add liquidity:", err)
						return
					}
				}
			case utils.RemoveLiquidity:
				r := rand.Intn(10000) % 2
				if r == 0 {
					// bep20-bep20
					pair, err := ea.GetPair(&utils.T_cfg.UniswapFactoryAddr, &utils.T_cfg.Bep20AddrsA[index], &utils.T_cfg.Bep20AddrsB[index])
					if err != nil {
						log.Println("error: get pair:", err)
						return
					}
					balance, err := ea.GetBEP20Balance(pair)
					if err != nil {
						log.Println("error: get bep20 balance:", err)
						return
					}
					// remove 1% liquidity
					amount := new(big.Int)
					amount.Div(balance, big.NewInt(100))
					log.Println("[debug]", balance, amount)
					_, err = ea.ApproveBEP20(nonce, pair, &utils.T_cfg.UniswapRouterAddr, amount)
					if err != nil {
						log.Println("error: approve bep20:", err)
						return
					}
					nonce++
					hash, err = ea.RemoveLiquidity(nonce, &utils.T_cfg.UniswapRouterAddr, &utils.T_cfg.Bep20AddrsA[index], &utils.T_cfg.Bep20AddrsB[index], amount, ea.Addr)
					if err != nil {
						log.Println("error: remove liquidity:", err)
						return
					}
				}
				if r == 1 {
					// wbnb-bep20
					pair, err := ea.GetPair(&utils.T_cfg.UniswapFactoryAddr, &utils.T_cfg.WbnbAddr, &utils.T_cfg.Bep20AddrsA[index])
					if err != nil {
						log.Println("error: get pair:", err, "bep20:", utils.T_cfg.Bep20AddrsA[index].Hex())
						return
					}
					balance, err := ea.GetBEP20Balance(pair)
					if err != nil {
						log.Println("error: get bep20 balance:", err)
						return
					}
					// remove 1% liquidity
					amount := new(big.Int)
					amount.Div(balance, big.NewInt(100))
					log.Println("[debug]", balance, amount)
					_, err = ea.ApproveBEP20(nonce, pair, &utils.T_cfg.UniswapRouterAddr, amount)
					if err != nil {
						log.Println("error: approve bep20:", err)
						return
					}
					nonce++
					hash, err = ea.RemoveLiquidity(nonce, &utils.T_cfg.UniswapRouterAddr, &utils.T_cfg.WbnbAddr, &utils.T_cfg.Bep20AddrsA[index], amount, ea.Addr)
					if err != nil {
						log.Println("error: remove liquidity:", err)
						return
					}
				}
			case utils.SwapExactTokensForTokens:
				path := make([]common.Address, 0, 2)
				r := rand.Intn(10000) % 2
				if r == 0 {
					path = append(path, utils.T_cfg.Bep20AddrsA[index])
					path = append(path, utils.T_cfg.Bep20AddrsB[index])
				}
				if r == 1 {
					path = append(path, utils.T_cfg.Bep20AddrsB[index])
					path = append(path, utils.T_cfg.Bep20AddrsA[index])
				}
				hash, err = ea.SwapExactTokensForTokens(nonce, &utils.T_cfg.UniswapRouterAddr, utils.T_cfg.LiquidityTestAmount, path, ea.Addr)
				if err != nil {
					log.Println("error: swap exact tokens for tokens:", err, path[0].Hex(), path[1].Hex())
					return
				}
			case utils.SwapBNBForExactTokens:
				// 50% wbnb will be returned
				actualAmount := new(big.Int)
				actualAmount.Div(utils.T_cfg.LiquidityTestAmount, big.NewInt(2))
				//
				path := make([]common.Address, 0, 2)
				r := rand.Intn(10000) % 2
				if r == 0 {
					path = append(path, utils.T_cfg.Bep20AddrsA[index])
					path = append(path, utils.T_cfg.WbnbAddr)
					hash, err = ea.SwapExactTokensForBNB(nonce, &utils.T_cfg.UniswapRouterAddr, utils.T_cfg.LiquidityTestAmount, actualAmount, path, ea.Addr)
					if err != nil {
						log.Println("error: SwapExactTokensForBNB:", err, path[0].Hex(), path[1].Hex())
						return
					}
				}
				if r == 1 {
					path = append(path, utils.T_cfg.WbnbAddr)
					path = append(path, utils.T_cfg.Bep20AddrsA[index])
					hash, err = ea.SwapBNBForExactTokens(nonce, &utils.T_cfg.UniswapRouterAddr, utils.T_cfg.LiquidityTestAmount, actualAmount, path, ea.Addr)
					if err != nil {
						log.Println("error: SwapBNBForExactTokens:", err, path[0].Hex(), path[1].Hex())
						return
					}
				}
			case utils.DepositWBNB:
				hash, err = ea.DepositWBNB(nonce, &utils.T_cfg.WbnbAddr, utils.T_cfg.LiquidityTestAmount)
				if err != nil {
					log.Println("error: deposit wbnb:", err)
					return
				}
			case utils.WithdrawWBNB:
				hash, err = ea.WithdrawWBNB(nonce, &utils.T_cfg.WbnbAddr, utils.T_cfg.LiquidityTestAmount)
				if err != nil {
					log.Println("error: withdraw wbnb:", err)
					return
				}
			case utils.ERC721MintOrTransfer:
				subScenario := utils.RandScenario(utils.T_cfg.ERC721MintOrTransfer)
				if subScenario.Name == utils.ERC721Mint {
					hash, err = ea.MintERC721(nonce, utils.T_cfg.Erc721Addr)
					if err != nil {
						log.Println("error: erc721Mint:", err)
						return
					}
				} else {
					tokenID, err := ea.GetOneERC721TokenID(utils.T_cfg.Erc721Addr)
					if err != nil {
						log.Println("error: get erc721 tokenID:", err)
						hash, err = ea.MintERC721(nonce, utils.T_cfg.Erc721Addr)
						if err != nil {
							log.Println("error: erc721Mint:", err)
							return
						}
					} else {
						_, err = ea.ApproveERC721(nonce, utils.T_cfg.Erc721Addr, randomAddress.Addr, tokenID)
						if err != nil {
							log.Println("error: approve erc721: ", err, randomAddress.Addr.String())
							return
						}
						nonce++
						hash, err = ea.TransferERC721(nonce, utils.T_cfg.Erc721Addr, randomAddress.Addr, tokenID)
						if err != nil {
							log.Println("error: transfer erc721: ", err)
							return
						}
					}
				}
			case utils.ERC1155MintOrBurnOrTransfer:
				switch utils.RandScenario(utils.T_cfg.ERC1155MintOrBurnOrTransfer).Name {
				case utils.ERC1155Mint:
					randomTokenID := rand.Int63n(utils.T_cfg.Erc1155InitTokenTypeNumber)
					hash, err = ea.MintERC1155(nonce, utils.T_cfg.Erc1155Addr, big.NewInt(randomTokenID), big.NewInt(utils.T_cfg.Erc1155InitTokenNumber))
					if err != nil {
						log.Println("error: erc1155 Mint:", err)
						return
					}
				case utils.ERC1155Burn:
					id, err := ea.GetOneERC1155TokenID(utils.T_cfg.Erc1155Addr, utils.T_cfg.Erc1155TokenIDSlice)
					if err != nil {
						log.Println("error: get erc1155 tokenID:", err)
						randomTokenID := rand.Int63n(utils.T_cfg.Erc1155InitTokenTypeNumber)
						hash, err = ea.MintERC1155(nonce, utils.T_cfg.Erc1155Addr, big.NewInt(randomTokenID), big.NewInt(utils.T_cfg.Erc1155InitTokenNumber))
						if err != nil {
							log.Println("error: erc1155 Mint:", err)
							return
						}
					} else {
						hash, err = ea.BurnERC1155(nonce, utils.T_cfg.Erc1155Addr, id, big.NewInt(utils.T_cfg.Erc1155InitTokenNumber))
						if err != nil {
							log.Println("error: erc1155 Burn:", err)
							return
						}
					}
				case utils.ERC1155Transfer:
					id, err := ea.GetOneERC1155TokenID(utils.T_cfg.Erc1155Addr, utils.T_cfg.Erc1155TokenIDSlice)
					if err != nil {
						log.Println("error: get erc1155 tokenID:", err)
						randomTokenID := rand.Int63n(utils.T_cfg.Erc1155InitTokenTypeNumber)
						hash, err = ea.MintERC1155(nonce, utils.T_cfg.Erc1155Addr, big.NewInt(randomTokenID), big.NewInt(utils.T_cfg.Erc1155InitTokenNumber))
						if err != nil {
							log.Println("error: erc1155 Mint:", err)
							return
						}
					} else {
						hash, err = ea.TransERC1155(nonce, utils.T_cfg.Erc1155Addr, *randomAddress.Addr, id, big.NewInt(utils.T_cfg.Erc1155InitTokenNumber))
						if err != nil {
							log.Println("error: erc1155 Trans:", err)
							return
						}
					}
				}
			}
			//
			m.Lock()
			results = append(results, hash)
			m.Unlock()
			//
		}(&wg, i, &eaSlice[i], &randomAddress)
		i++
		if i == len(eaSlice) {
			i = 0
		}
	}
	wg.Wait()
	return results
}

func checkAllTransactionStatus(root *utils.ExtAcc, hashList []*common.Hash) int {
	var wg sync.WaitGroup
	var numberLock sync.Mutex
	wg.Add(len(hashList))
	limiter := ratelimit.New(*tps)
	txnFinishedNumber := 0
	for i := 0; i < len(hashList); i++ {
		limiter.Take()
		receipt := root.GetReceipt(hashList[i], 10)
		if receipt.Status == 1 {
			numberLock.Lock()
			txnFinishedNumber++
			numberLock.Unlock()
		}
	}
	return txnFinishedNumber
}

func setupTimer(dur time.Duration) *bool {
	t := time.NewTimer(dur)
	expired := false
	go func() {
		<-t.C
		expired = true
	}()
	return &expired
}
