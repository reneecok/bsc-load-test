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

var endpoints *string
var fullnodes []string

var roothexkey *string
var roothexaddr *string
var hexkeyfile *string

var usersCreated *int
var usersLoaded *int

var randTestAcc *bool
var initTestAcc *bool
var resetTestAcc *bool

var bep20Hex *string
// A tokens for liquidity pairs
var bep20AddrsA []common.Address
// B tokens for liquidity pairs
var bep20AddrsB []common.Address

var wbnbHex *string
var wbnbAddr common.Address

var uniswapFactoryHex *string
var uniswapFactoryAddr common.Address
var uniswapRouterHex *string
var uniswapRouterAddr common.Address

var runTestAcc *bool
var scenarios []utils.Scenario

var tps *int
var sec *int

var queryBlocks *bool
var blockNumS *int64
var blockNumE *int64

var debug *bool

var distributeAmount *big.Int
var liquidityInitAmount *big.Int
var liquidityTestAmount *big.Int

func init() {
	//
	rand.Seed(time.Now().UnixNano())
	//
	endpoints = flag.String("endpoints", "", "endpoints")
	roothexkey = flag.String("roothexkey", "", "roothexkey")
	roothexaddr = flag.String("roothexaddr", "", "roothexaddr")
	hexkeyfile = flag.String("hexkeyfile", "", "hexkeyfile")
	usersCreated = flag.Int("usersCreated", 50000, "usersCreated")
	usersLoaded = flag.Int("usersLoaded", 50000, "usersLoaded")
	randTestAcc = flag.Bool("randTestAcc", false, "randTestAcc")
	initTestAcc = flag.Bool("initTestAcc", false, "initTestAcc")
	resetTestAcc = flag.Bool("resetTestAcc", false, "resetTestAcc")
	bep20Hex = flag.String("bep20Hex", "", "bep20Hex")
	wbnbHex = flag.String("wbnbHex", "", "wbnbHex")
	uniswapFactoryHex = flag.String("uniswapFactoryHex", "", "uniswapFactoryHex")
	uniswapRouterHex = flag.String("uniswapRouterHex", "", "uniswapRouterHex")
	runTestAcc = flag.Bool("runTestAcc", false, "runTestAcc")
	tps = flag.Int("tps", 1, "tps")
	sec = flag.Int("sec", 10, "sec")
	queryBlocks = flag.Bool("queryBlocks", false, "queryBlocks")
	blockNumS = flag.Int64("blockNumS", 0, "blockNumS")
	blockNumE = flag.Int64("blockNumE", 0, "blockNumE")
	debug = flag.Bool("debug", false, "debug")

	flag.Parse()

	fullnodes = strings.Split(*endpoints, ",")

	tokens := strings.Split(*bep20Hex, ",")

	h := len(tokens)/2
	l := len(tokens)
	for _, v := range tokens[0:h] {
		bep20AddrsA = append(bep20AddrsA, common.HexToAddress(v))
	}
	for _, v := range tokens[h:l] {
		bep20AddrsB = append(bep20AddrsB, common.HexToAddress(v))
	}
	if *bep20Hex != "" && len(bep20AddrsA) != len(bep20AddrsB) {
		panic("unbalanced bep20 pair(s) found")
	}

	wbnbAddr = common.HexToAddress(*wbnbHex)

	uniswapFactoryAddr = common.HexToAddress(*uniswapFactoryHex)
	uniswapRouterAddr = common.HexToAddress(*uniswapRouterHex)

	scenarios = []utils.Scenario {
		utils.Scenario{utils.SendBNB, 5},
		utils.Scenario{utils.SendBEP20, 5},
		utils.Scenario{utils.AddLiquidity, 5},
		utils.Scenario{utils.RemoveLiquidity, 5},
		utils.Scenario{utils.SwapExactTokensForTokens, 35},
		utils.Scenario{utils.SwapBNBForExactTokens, 35},
		utils.Scenario{utils.DepositWBNB, 5},
		utils.Scenario{utils.WithdrawWBNB, 5},
		}

	distributeAmount = big.NewInt(1e18)
	liquidityInitAmount = new(big.Int)
	liquidityTestAmount = new(big.Int)
	liquidityInitAmount.Div(distributeAmount, big.NewInt(4))
	liquidityTestAmount.Div(liquidityInitAmount, big.NewInt(2.5e12))

	log.Println("distributeAmount:", distributeAmount)
	log.Println("liquidityInitAmount:", liquidityInitAmount)
	log.Println("liquidityTestAmount:", liquidityTestAmount)
}

func main() {
	//
	if *randTestAcc {
		utils.RandHexKeys(*hexkeyfile, *usersCreated)
		return
	}
	//
	clients := make([]*ethclient.Client, 0, len(fullnodes))
	for i, v := range fullnodes {
		log.Printf("%d: %s", i, v)
		client, err := ethclient.Dial(v)
		if err != nil {
			panic(err)
		}
		clients = append(clients, client)
	}
	defer cleanup(clients)
	//
	root, err := utils.NewExtAcc(clients[0],
		*roothexkey, *roothexaddr)
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
	if *bep20Hex != "" {
		for i, _ := range bep20AddrsA {
			_, err = root.GetBEP20Balance(&bep20AddrsA[i])
			if err != nil {
				panic(err)
			}
			_, err = root.GetBEP20Balance(&bep20AddrsB[i])
			if err != nil {
				panic(err)
			}
		}
	}
	if *wbnbHex != "" {
		_, err := root.GetBEP20Balance(&wbnbAddr)
		if err != nil {
			panic(err)
		}
	}
	//
	nonce, err := root.Client.PendingNonceAt(
		context.Background(), *root.Addr)
	if err != nil {
		panic(err)
	}
	log.Println("root: nonce -", nonce)
	//
	if *initTestAcc {
		//
		limiter := ratelimit.New(*tps)
		//
		eaSlice := load(clients)
		//
		for i, v := range eaSlice {
			limiter.Take()
			//
			_, err := root.SendBNB(nonce, v.Addr, distributeAmount)
			if err != nil {
				log.Println("error: send bnb:", err)
				continue
			}
			nonce++
			//
			if *bep20Hex != "" {
				//
				index := i % len(bep20AddrsA)
				//
				_, err = root.SendBEP20(nonce, &bep20AddrsA[index], v.Addr, distributeAmount)
				if err != nil {
					log.Println("error: send bep20:", err)
					continue
				}
				nonce++
				//
				_, err = root.SendBEP20(nonce, &bep20AddrsB[index], v.Addr, distributeAmount)
				if err != nil {
					log.Println("error: send bep20:", err)
					continue
				}
				nonce++
			}
		}
		time.Sleep(10 * time.Second)
		//
		if *wbnbHex != "" && *uniswapFactoryHex != "" && *uniswapRouterHex != "" {
			//
			var wg sync.WaitGroup
			wg.Add(len(eaSlice))
			for i, v := range eaSlice {
				limiter.Take()
				go func(wg *sync.WaitGroup, i int, ea utils.ExtAcc) {
					defer wg.Done()
					//
					index := i % len(bep20AddrsA)
					err = initUniswapByAcc(&ea, &bep20AddrsA[index], &bep20AddrsB[index])
					if err != nil {
						log.Println("error: initUniswapByAcc:", err)
						return
					}
				}(&wg, i, v)
			}
			wg.Wait()
		}
		return
	}
	//
	if *resetTestAcc {
		//
		limiter := ratelimit.New(*tps)
		//
		var wg sync.WaitGroup
		eaSlice := load(clients)
		//
		if *wbnbHex != "" {
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
					balance, err := ea.GetBEP20Balance(&wbnbAddr)
					if err != nil {
						log.Println("error: get wbnb balance: %s, %s\n",
							ea.Addr.Hex(), err)
						return
					}
					_, err = ea.WithdrawWBNB(nonce, &wbnbAddr, balance)
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
				nonce, err := ea.Client.PendingNonceAt(
					context.Background(), *ea.Addr)
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
		eaSlice := load(clients)
		block, err := root.Client.BlockByNumber(
			context.Background(), nil)
		if err != nil {
			panic(err)
		}
		log.Printf("the latest block: %d\n", block.Number().Uint64())
		results := exec(eaSlice)
		log.Println("# of tx hash returned in load test:", len(results))
		dir := filepath.Dir(*hexkeyfile)
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

func load(clients []*ethclient.Client) []utils.ExtAcc {
	items := utils.LoadHexKeys(*hexkeyfile, *usersLoaded)
	eaSlice := make([]utils.ExtAcc, 0, len(items))
	for i, v := range items {
		client := clients[i%len(clients)]
		ea, err := utils.NewExtAcc(client, v[0], v[1])
		if err != nil {
			panic(err.Error())
		}
		eaSlice = append(eaSlice, *ea)
	}
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
	wbnbAmount.Mul(liquidityInitAmount, big.NewInt(2))
	_, err = acc.DepositWBNB(nonce, &wbnbAddr, wbnbAmount)
	if err != nil {
		log.Println("error: deposit wbnb: " + err.Error())
		return err
	}
	nonce++
	_, err = acc.ApproveBEP20(nonce, &wbnbAddr, &uniswapRouterAddr, distributeAmount)
	if err != nil {
		log.Println("error: approve wbnb: " + err.Error())
		return err
	}
	nonce++
	_, err = acc.ApproveBEP20(nonce, tokenA, &uniswapRouterAddr, distributeAmount)
	if err != nil {
		log.Println("error: approve bep20: " + err.Error())
		return err
	}
	nonce++
	_, err = acc.ApproveBEP20(nonce, tokenB, &uniswapRouterAddr, distributeAmount)
	if err != nil {
		log.Println("error: approve bep20: " + err.Error())
		return err
	}
	nonce++
	_, err = acc.AddLiquidity(nonce, &uniswapRouterAddr, &wbnbAddr, tokenA, liquidityInitAmount, liquidityInitAmount, acc.Addr)
	if err != nil {
		log.Println("error: add liquidity: " + err.Error())
		return err
	}
	nonce++
	_, err = acc.AddLiquidity(nonce, &uniswapRouterAddr, tokenA, tokenB, liquidityInitAmount, liquidityInitAmount, acc.Addr)
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
		if *expired {
			break
		}
		wg.Add(1)
		go func(wg *sync.WaitGroup, i int, ea *utils.ExtAcc) {
			defer wg.Done()
			//
			scenario := utils.RandScenario(scenarios)
			//
			nonce, err := ea.Client.PendingNonceAt(
				context.Background(), *ea.Addr)
			if err != nil {
				log.Println("error: nonce:", err)
				return
			}
			//
			j := rand.Intn(*usersLoaded)
			index := i % len(bep20AddrsA)
			//
			var hash *common.Hash
			//
			if scenario.Name == utils.SendBNB {
				//
				hash, err = ea.SendBNB(nonce, eaSlice[j].Addr, liquidityTestAmount)
				if err != nil {
					log.Println("error: send bnb:", err)
					return
				}
			} else if scenario.Name == utils.SendBEP20 {
				//
				hash, err = ea.SendBEP20(nonce, &bep20AddrsA[index], eaSlice[j].Addr, liquidityTestAmount)
				if err != nil {
					log.Println("error: send bep20:", err)
						return
				}
			} else if scenario.Name == utils.AddLiquidity {
				//
				r := rand.Intn(2)
				if r == 0 {
					// bep20-bep20
					_, err := ea.ApproveBEP20(nonce, &bep20AddrsA[index], &uniswapRouterAddr, liquidityTestAmount)
					if err != nil {
						log.Println("error: approve bep20:", err)
						return
					}
					nonce++
					_, err = ea.ApproveBEP20(nonce, &bep20AddrsB[index], &uniswapRouterAddr, liquidityTestAmount)
					if err != nil {
						log.Println("error: approve bep20:", err)
						return
					}
					nonce++
					hash, err = ea.AddLiquidity(nonce, &uniswapRouterAddr, &bep20AddrsA[index], &bep20AddrsB[index], liquidityTestAmount, liquidityTestAmount, ea.Addr)
					if err != nil {
						log.Println("error: add liquidity:", err)
						return
					}
				}
				if r == 1 {
					// wbnb-bep20
					_, err := ea.ApproveBEP20(nonce, &wbnbAddr, &uniswapRouterAddr, liquidityTestAmount)
					if err != nil {
						log.Println("error: approve wbnb:", err)
						return
					}
					nonce++
					_, err = ea.ApproveBEP20(nonce, &bep20AddrsA[index], &uniswapRouterAddr, liquidityTestAmount)
					if err != nil {
						log.Println("error: approve bep20:", err)
						return
					}
					nonce++
					hash, err = ea.AddLiquidity(nonce, &uniswapRouterAddr, &wbnbAddr, &bep20AddrsA[index], liquidityTestAmount, liquidityTestAmount, ea.Addr)
					if err != nil {
						log.Println("error: add liquidity:", err)
						return
					}
				}
			} else if scenario.Name == utils.RemoveLiquidity {
				//
				r := rand.Intn(2)
				if r == 0 {
					// bep20-bep20
					pair, err := ea.GetPair(&uniswapFactoryAddr, &bep20AddrsA[index], &bep20AddrsB[index])
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
					_, err = ea.ApproveBEP20(nonce, pair, &uniswapRouterAddr, amount)
					if err != nil {
						log.Println("error: approve bep20:", err)
						return
					}
					nonce++
					hash, err = ea.RemoveLiquidity(nonce, &uniswapRouterAddr, &bep20AddrsA[index], &bep20AddrsB[index], amount, ea.Addr)
					if err != nil {
						log.Println("error: remove liquidity:", err)
						return
					}
				}
				if r == 1 {
					// wbnb-bep20
					pair, err := ea.GetPair(&uniswapFactoryAddr, &wbnbAddr, &bep20AddrsA[index])
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
					_, err = ea.ApproveBEP20(nonce, pair, &uniswapRouterAddr, amount)
					if err != nil {
						log.Println("error: approve bep20:", err)
						return
					}
					nonce++
					hash, err = ea.RemoveLiquidity(nonce, &uniswapRouterAddr, &wbnbAddr, &bep20AddrsA[index], amount, ea.Addr)
					if err != nil {
						log.Println("error: remove liquidity:", err)
						return
					}
				}
			} else if scenario.Name == utils.SwapExactTokensForTokens {
				//
				path := make([]common.Address, 0, 2)
				r := rand.Intn(2)
				if r == 0 {
					path = append(path, bep20AddrsA[index])
					path = append(path, bep20AddrsB[index])
				}
				if r == 1 {
					path = append(path, bep20AddrsB[index])
					path = append(path, bep20AddrsA[index])
				}
				hash, err = ea.SwapExactTokensForTokens(nonce, &uniswapRouterAddr, liquidityTestAmount, path, ea.Addr)
				if err != nil {
					log.Println("error: swap exact tokens for tokens:", err)
					return
				}
			} else if scenario.Name == utils.SwapBNBForExactTokens {
				//
				// 50% wbnb will be returned
				actualAmount := new(big.Int)
				actualAmount.Div(liquidityTestAmount, big.NewInt(2))
				//
				path := make([]common.Address, 0, 2)
				path = append(path, wbnbAddr)
				path = append(path, bep20AddrsA[index])
				//
				hash, err = ea.SwapBNBForExactTokens(nonce, &uniswapRouterAddr, liquidityTestAmount, actualAmount, path, ea.Addr)
				if err != nil {
					log.Println("error: swap bnb for exact tokens:", err)
					return
				}
			} else if scenario.Name == utils.DepositWBNB {
				//
				hash, err = ea.DepositWBNB(nonce, &wbnbAddr, liquidityTestAmount)
				if err != nil {
					log.Println("error: deposit wbnb:", err)
					return
				}
			} else if scenario.Name == utils.WithdrawWBNB  {
				//
				hash, err = ea.WithdrawWBNB(nonce, &wbnbAddr, liquidityTestAmount)
				if err != nil {
					log.Println("error: withdraw wbnb:", err)
					return
				}
			}
			//
			m.Lock()
			results = append(results, hash)
			m.Unlock()
			//
		}(&wg, i, &eaSlice[i])
		i++
		if i == len(eaSlice) {
			i = 0
		}
	}
	wg.Wait()
	//
	return results
}

func setupTimer(dur time.Duration) *bool {
	t := time.NewTimer(dur)
	expired := false
	go func() {
		<- t.C
		expired = true
	}()
	return &expired
}
