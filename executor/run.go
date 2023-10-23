package executor

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"bsc-load-test/log"
	"bsc-load-test/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/ratelimit"
)

func Run(clients []*ethclient.Client, root *utils.ExtAcc) {
	eaSlice := utils.Load(clients, utils.T_cfg.Hexkeyfile, &utils.T_cfg.UsersLoaded)

	block, err := root.Client.BlockByNumber(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	log.Printf("the latest block: %d\n", block.Number().Uint64())
	results := exec(eaSlice)
	utils.CheckAllTransactionStatus(root, results, utils.T_cfg.Tps)

	dir := filepath.Dir(utils.T_cfg.Hexkeyfile)
	suffix := time.Now().UnixNano()
	fullPath := filepath.Join(dir, "results", fmt.Sprintf("results_%d.csv", suffix))
	if err = utils.SaveHash(fullPath, results); err != nil {
		log.Printf("error: save tx hash to file failed, total %v", len(results))
	}
	return
}

func exec(eaSlice []utils.ExtAcc) []*common.Hash {
	limiter := ratelimit.New(utils.T_cfg.Tps)
	dur := time.Duration(utils.T_cfg.Sec) * time.Second
	expired := utils.SetupTimer(dur)
	trans := (utils.T_cfg.Tps) * (utils.T_cfg.Sec)
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
				if hash, err = ea.SendBNB(nonce, eaSlice[j].Addr, utils.T_cfg.LiquidityTestAmount); err != nil {
					log.Println("error: send bnb:", err)
					return
				}
			case utils.SendBEP20:
				if hash, err = ea.SendBEP20(nonce, &utils.T_cfg.Bep20AddrsA[index], eaSlice[j].Addr, utils.T_cfg.LiquidityTestAmount); err != nil {
					log.Println("error: send bep20:", err)
					return
				}
			case utils.AddLiquidity:
				if hash, err = AddLiquidity(ea, nonce, index); err != nil {
					log.Println("error: AddLiquidity:", err)
					return
				}
			case utils.RemoveLiquidity:
				if hash, err = RemoveLiquidity(ea, nonce, index); err != nil {
					log.Println("error: RemoveLiquidity:", err)
					return
				}
			case utils.SwapExactTokensForTokens:
				if hash, err = SwapExactTokensForTokens(ea, nonce, index); err != nil {
					log.Println("error: SwapExactTokensForTokens:", err)
					return
				}
			case utils.SwapBNBForExactTokens:
				if hash, err = SwapBNBForExactTokens(ea, nonce, index); err != nil {
					log.Println("error: SwapBNBForExactTokens:", err)
					return
				}
			case utils.DepositWBNB:
				if hash, err = ea.DepositWBNB(nonce, utils.T_cfg.LiquidityTestAmount); err != nil {
					log.Println("error: deposit wbnb:", err)
					return
				}
			case utils.WithdrawWBNB:
				if hash, err = ea.WithdrawWBNB(nonce, utils.T_cfg.LiquidityTestAmount); err != nil {
					log.Println("error: withdraw wbnb:", err)
					return
				}
			case utils.ERC721MintOrTransfer:
				if hash, err = ERC721MintOrTransfer(ea, nonce, randomAddress); err != nil {
					log.Println("error: ERC721MintOrTransfer:", err)
					return
				}
			case utils.ERC1155MintOrBurnOrTransfer:
				if hash, err = ERC1155MintOrBurnOrTransfer(ea, nonce, randomAddress); err != nil {
					log.Println("error: ERC1155MintOrBurnOrTransfer:", err)
					return
				}
			}
			//
			m.Lock()
			results = append(results, hash)
			m.Unlock()
		}(&wg, i, &eaSlice[i], &randomAddress)
		i++
		if i == len(eaSlice) {
			i = 0
		}
	}
	wg.Wait()
	return results
}
