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
var erc721Hex, erc1155Hex *string
var erc721Addr, erc1155Addr common.Address

var runTestAcc *bool
var scenarios []utils.Scenario
var erc721MintOrTransfer []utils.Scenario
var erc1155MintOrBurnOrTransfer []utils.Scenario

var tps *int
var sec *int

var queryBlocks *bool
var blockNumS *int64
var blockNumE *int64

var debug *bool

var distributeAmount *big.Int
var liquidityInitAmount *big.Int
var liquidityTestAmount *big.Int
var erc721InitTokenNumber int
var erc1155InitTokenTypeNumber, erc1155InitTokenNumber int64
var erc1155TokenIDSlice []*big.Int

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
	erc721Hex = flag.String("erc721Hex", "", "erc721Hex")
	erc1155Hex = flag.String("erc1155Hex", "", "erc1155Hex")
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

	h := len(tokens) / 2
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
	erc721Addr = common.HexToAddress(*erc721Hex)
	erc1155Addr = common.HexToAddress(*erc1155Hex)

	scenarios = []utils.Scenario{
		{utils.SendBNB, 0},
		{utils.SendBEP20, 0},
		{utils.AddLiquidity, 0},
		{utils.RemoveLiquidity, 0},
		{utils.SwapExactTokensForTokens, 0},
		{utils.SwapBNBForExactTokens, 0},
		{utils.DepositWBNB, 0},
		{utils.WithdrawWBNB, 0},
		{utils.ERC721MintOrTransfer, 0},
		{utils.ERC1155MintOrBurnOrTransfer, 100},
	}
	erc721MintOrTransfer = []utils.Scenario{
		{utils.ERC721Mint, 1},
		{utils.ERC721Transfer, 9},
	}
	erc1155MintOrBurnOrTransfer = []utils.Scenario{
		{utils.ERC1155Mint, 6},
		{utils.ERC1155Burn, 3},
		{utils.ERC1155Transfer, 1},
	}
	erc721InitTokenNumber = 2
	erc1155InitTokenTypeNumber = 30
	erc1155InitTokenNumber = 5
	distributeAmount = big.NewInt(1e18)
	liquidityInitAmount = new(big.Int)
	liquidityTestAmount = new(big.Int)
	liquidityInitAmount.Div(distributeAmount, big.NewInt(4))
	liquidityTestAmount.Div(liquidityInitAmount, big.NewInt(2.5e12))
	for i := int64(0); i < erc1155InitTokenTypeNumber; i++ {
		erc1155TokenIDSlice = append(erc1155TokenIDSlice, big.NewInt(i))
	}

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
		for i := range bep20AddrsA {
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
			_, err = root.SendBNB(nonce, v.Addr, distributeAmount)
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
		time.Sleep(10 * time.Second)

		if *erc721Hex != "" {
			var wg sync.WaitGroup
			var totalAccountSlice []utils.ExtAcc
			for i := 0; i < erc721InitTokenNumber; i++ {
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
					_, err = v.MintERC721(nonce, erc721Addr)
					if err != nil {
						log.Println("error: mint erc721:", err)
						return
					}
				}(&wg, v)
			}
			wg.Wait()
		}

		if *erc1155Hex != "" {
			var wg sync.WaitGroup
			wg.Add(len(eaSlice))
			var tokenAmountSlice []*big.Int
			for range erc1155TokenIDSlice {
				tokenAmountSlice = append(tokenAmountSlice, big.NewInt(erc1155InitTokenNumber))
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
					_, err = v.MintBatchERC1155(nonce, erc1155Addr, erc1155TokenIDSlice, tokenAmountSlice)
					if err != nil {
						log.Println("error: mint batch erc1155:", err)
						return
					}
				}(&wg, v)
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
		eaSlice := load(clients)
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
	batches := utils.LoadHexKeys(*hexkeyfile, *usersLoaded)
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
		randomAddress := eaSlice[rand.Intn(len(eaSlice))]
		if *expired {
			break
		}
		wg.Add(1)
		go func(wg *sync.WaitGroup, i int, ea, randomAddress *utils.ExtAcc) {
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
			switch scenario.Name {
			case utils.SendBNB:
				hash, err = ea.SendBNB(nonce, eaSlice[j].Addr, liquidityTestAmount)
				if err != nil {
					log.Println("error: send bnb:", err)
					return
				}
			case utils.SendBEP20:
				hash, err = ea.SendBEP20(nonce, &bep20AddrsA[index], eaSlice[j].Addr, liquidityTestAmount)
				if err != nil {
					log.Println("error: send bep20:", err)
					return
				}
			case utils.AddLiquidity:
				r := rand.Intn(10000) % 2
				if r == 0 {
					// bep20-bep20
					_, err := ea.ApproveBEP20(nonce, &bep20AddrsA[index], &uniswapRouterAddr, distributeAmount)
					if err != nil {
						log.Println("error: approve bep20:", err)
						return
					}
					nonce++
					_, err = ea.ApproveBEP20(nonce, &bep20AddrsB[index], &uniswapRouterAddr, distributeAmount)
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
					_, err := ea.ApproveBEP20(nonce, &wbnbAddr, &uniswapRouterAddr, distributeAmount)
					if err != nil {
						log.Println("error: approve wbnb:", err)
						return
					}
					nonce++
					_, err = ea.ApproveBEP20(nonce, &bep20AddrsA[index], &uniswapRouterAddr, distributeAmount)
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
			case utils.RemoveLiquidity:
				r := rand.Intn(10000) % 2
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
						log.Println("error: get pair:", err, "bep20:", bep20AddrsA[index].Hex())
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
			case utils.SwapExactTokensForTokens:
				path := make([]common.Address, 0, 2)
				r := rand.Intn(10000) % 2
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
					log.Println("error: swap exact tokens for tokens:", err, path[0].Hex(), path[1].Hex())
					return
				}
			case utils.SwapBNBForExactTokens:
				// 50% wbnb will be returned
				actualAmount := new(big.Int)
				actualAmount.Div(liquidityTestAmount, big.NewInt(2))
				//
				path := make([]common.Address, 0, 2)
				r := rand.Intn(10000) % 2
				if r == 0 {
					path = append(path, bep20AddrsA[index])
					path = append(path, wbnbAddr)
					hash, err = ea.SwapExactTokensForBNB(nonce, &uniswapRouterAddr, liquidityTestAmount, actualAmount, path, ea.Addr)
					if err != nil {
						log.Println("error: SwapExactTokensForBNB:", err, path[0].Hex(), path[1].Hex())
						return
					}
				}
				if r == 1 {
					path = append(path, wbnbAddr)
					path = append(path, bep20AddrsA[index])
					hash, err = ea.SwapBNBForExactTokens(nonce, &uniswapRouterAddr, liquidityTestAmount, actualAmount, path, ea.Addr)
					if err != nil {
						log.Println("error: SwapBNBForExactTokens:", err, path[0].Hex(), path[1].Hex())
						return
					}
				}
			case utils.DepositWBNB:
				hash, err = ea.DepositWBNB(nonce, &wbnbAddr, liquidityTestAmount)
				if err != nil {
					log.Println("error: deposit wbnb:", err)
					return
				}
			case utils.WithdrawWBNB:
				hash, err = ea.WithdrawWBNB(nonce, &wbnbAddr, liquidityTestAmount)
				if err != nil {
					log.Println("error: withdraw wbnb:", err)
					return
				}
			case utils.ERC721MintOrTransfer:
				subScenario := utils.RandScenario(erc721MintOrTransfer)
				if subScenario.Name == utils.ERC721Mint {
					hash, err = ea.MintERC721(nonce, erc721Addr)
					if err != nil {
						log.Println("error: erc721Mint:", err)
						return
					}
				} else {
					tokenID, err := ea.GetOneERC721TokenID(erc721Addr)
					if err != nil {
						log.Println("error: get erc721 tokenID:", err)
						hash, err = ea.MintERC721(nonce, erc721Addr)
						if err != nil {
							log.Println("error: erc721Mint:", err)
							return
						}
					} else {
						_, err = ea.ApproveERC721(nonce, erc721Addr, randomAddress.Addr, tokenID)
						if err != nil {
							log.Println("error: approve erc721: ", err, randomAddress.Addr.String())
							return
						}
						nonce++
						hash, err = ea.TransferERC721(nonce, erc721Addr, randomAddress.Addr, tokenID)
						if err != nil {
							log.Println("error: transfer erc721: ", err)
							return
						}
					}
				}
			case utils.ERC1155MintOrBurnOrTransfer:
				switch utils.RandScenario(erc1155MintOrBurnOrTransfer).Name {
				case utils.ERC1155Mint:
					randomTokenID := rand.Int63n(erc1155InitTokenTypeNumber)
					hash, err = ea.MintERC1155(nonce, erc1155Addr, big.NewInt(randomTokenID), big.NewInt(erc1155InitTokenNumber))
					if err != nil {
						log.Println("error: erc1155 Mint:", err)
						return
					}
				case utils.ERC1155Burn:
					id, err := ea.GetOneERC1155TokenID(erc1155Addr, erc1155TokenIDSlice)
					if err != nil {
						log.Println("error: get erc1155 tokenID:", err)
						randomTokenID := rand.Int63n(erc1155InitTokenTypeNumber)
						hash, err = ea.MintERC1155(nonce, erc1155Addr, big.NewInt(randomTokenID), big.NewInt(erc1155InitTokenNumber))
						if err != nil {
							log.Println("error: erc1155 Mint:", err)
							return
						}
					} else {
						hash, err = ea.BurnERC1155(nonce, erc1155Addr, id, big.NewInt(erc1155InitTokenNumber))
						if err != nil {
							log.Println("error: erc1155 Burn:", err)
							return
						}
					}
				case utils.ERC1155Transfer:
					id, err := ea.GetOneERC1155TokenID(erc1155Addr, erc1155TokenIDSlice)
					if err != nil {
						log.Println("error: get erc1155 tokenID:", err)
						randomTokenID := rand.Int63n(erc1155InitTokenTypeNumber)
						hash, err = ea.MintERC1155(nonce, erc1155Addr, big.NewInt(randomTokenID), big.NewInt(erc1155InitTokenNumber))
						if err != nil {
							log.Println("error: erc1155 Mint:", err)
							return
						}
					} else {
						hash, err = ea.TransERC1155(nonce, erc1155Addr, *randomAddress.Addr, id, big.NewInt(erc1155InitTokenNumber))
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
