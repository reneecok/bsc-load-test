package executor

import (
	"context"
	"log"
	"math/big"
	"sync"
	"time"

	"bsc-load-test/utils"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/ratelimit"
)

func ResetTest(clients []*ethclient.Client, nonce uint64, root *utils.ExtAcc) {
	eaSlice := utils.Load(clients, utils.T_cfg.Hexkeyfile, &utils.T_cfg.UsersLoaded)
	limiter := ratelimit.New(utils.T_cfg.Tps)
	var wg sync.WaitGroup
	var err error
	//
	if utils.T_cfg.WbnbHex != "" {
		wg.Add(len(eaSlice))
		for _, ea := range eaSlice {
			limiter.Take()
			go func(wg *sync.WaitGroup, ea utils.ExtAcc) {
				defer wg.Done()
				//
				nonce, err = ea.Client.PendingNonceAt(context.Background(), *ea.Addr)
				if err != nil {
					log.Println("error: nonce:", err)
					return
				}
				//
				balance, err := ea.GetBEP20Balance(&utils.T_cfg.WbnbAddr)
				if err != nil {
					log.Printf("error: get wbnb balance: %s, %s", ea.Addr.Hex(), err)
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
			nonce, err = ea.Client.PendingNonceAt(context.Background(), *ea.Addr)
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
