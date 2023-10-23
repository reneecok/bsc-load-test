package executor

import (
	"math/big"
	"math/rand"

	"bsc-load-test/log"
	"bsc-load-test/utils"

	"github.com/ethereum/go-ethereum/common"
)

func AddLiquidity(ea *utils.ExtAcc, nonce uint64, index int) (*common.Hash, error) {
	var hash *common.Hash
	r := rand.Intn(10000) % 2
	if r == 0 {
		// bep20-bep20
		_, err := ea.ApproveBEP20(nonce, &utils.T_cfg.Bep20AddrsA[index], &utils.T_cfg.UniswapRouterAddr, utils.T_cfg.DistributeAmount)
		if err != nil {
			log.Println("error: approve bep20:", err)
			return nil, err
		}
		nonce++
		_, err = ea.ApproveBEP20(nonce, &utils.T_cfg.Bep20AddrsB[index], &utils.T_cfg.UniswapRouterAddr, utils.T_cfg.DistributeAmount)
		if err != nil {
			log.Println("error: approve bep20:", err)
			return nil, err
		}
		nonce++
		hash, err = ea.AddLiquidity(nonce, &utils.T_cfg.Bep20AddrsA[index], &utils.T_cfg.Bep20AddrsB[index],
			utils.T_cfg.LiquidityTestAmount, utils.T_cfg.LiquidityTestAmount, ea.Addr)
		if err != nil {
			log.Println("error: add liquidity:", err)
			return nil, err
		}
	}
	if r == 1 {
		// wbnb-bep20
		_, err := ea.ApproveBEP20(nonce, &utils.T_cfg.WbnbAddr, &utils.T_cfg.UniswapRouterAddr, utils.T_cfg.DistributeAmount)
		if err != nil {
			log.Println("error: approve wbnb:", err)
			return nil, err
		}
		nonce++
		_, err = ea.ApproveBEP20(nonce, &utils.T_cfg.Bep20AddrsA[index], &utils.T_cfg.UniswapRouterAddr, utils.T_cfg.DistributeAmount)
		if err != nil {
			log.Println("error: approve bep20:", err)
			return nil, err
		}
		nonce++
		hash, err = ea.AddLiquidity(nonce, &utils.T_cfg.WbnbAddr, &utils.T_cfg.Bep20AddrsA[index],
			utils.T_cfg.LiquidityTestAmount, utils.T_cfg.LiquidityTestAmount, ea.Addr)
		if err != nil {
			log.Println("error: add liquidity:", err)
			return nil, err
		}
	}
	return hash, nil
}

func RemoveLiquidity(ea *utils.ExtAcc, nonce uint64, index int) (*common.Hash, error) {
	var hash *common.Hash
	r := rand.Intn(10000) % 2
	if r == 0 {
		// bep20-bep20
		pair, err := ea.GetPair(&utils.T_cfg.Bep20AddrsA[index], &utils.T_cfg.Bep20AddrsB[index])
		if err != nil {
			log.Println("error: get pair:", err)
			return nil, err
		}
		balance, err := ea.GetBEP20Balance(pair)
		if err != nil {
			log.Println("error: get bep20 balance:", err)
			return nil, err
		}
		// remove 1% liquidity
		amount := new(big.Int)
		amount.Div(balance, big.NewInt(100))
		log.Println("[debug]", balance, amount)
		_, err = ea.ApproveBEP20(nonce, pair, &utils.T_cfg.UniswapRouterAddr, amount)
		if err != nil {
			log.Println("error: approve bep20:", err)
			return nil, err
		}
		nonce++
		hash, err = ea.RemoveLiquidity(nonce, &utils.T_cfg.Bep20AddrsA[index], &utils.T_cfg.Bep20AddrsB[index], amount, ea.Addr)
		if err != nil {
			log.Println("error: remove liquidity:", err)
			return nil, err
		}
	}
	if r == 1 {
		// wbnb-bep20
		pair, err := ea.GetPair(&utils.T_cfg.WbnbAddr, &utils.T_cfg.Bep20AddrsA[index])
		if err != nil {
			log.Println("error: get pair:", err, "bep20:", utils.T_cfg.Bep20AddrsA[index].Hex())
			return nil, err
		}
		balance, err := ea.GetBEP20Balance(pair)
		if err != nil {
			log.Println("error: get bep20 balance:", err)
			return nil, err
		}
		// remove 1% liquidity
		amount := new(big.Int)
		amount.Div(balance, big.NewInt(100))
		log.Println("[debug]", balance, amount)
		_, err = ea.ApproveBEP20(nonce, pair, &utils.T_cfg.UniswapRouterAddr, amount)
		if err != nil {
			log.Println("error: approve bep20:", err)
			return nil, err
		}
		nonce++
		hash, err = ea.RemoveLiquidity(nonce, &utils.T_cfg.WbnbAddr, &utils.T_cfg.Bep20AddrsA[index], amount, ea.Addr)
		if err != nil {
			log.Println("error: remove liquidity:", err)
			return nil, err
		}
	}
	return hash, nil
}

func SwapExactTokensForTokens(ea *utils.ExtAcc, nonce uint64, index int) (*common.Hash, error) {
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
	hash, err := ea.SwapExactTokensForTokens(nonce, utils.T_cfg.LiquidityTestAmount, path, ea.Addr)
	if err != nil {
		log.Println("error: swap exact tokens for tokens:", err, path[0].Hex(), path[1].Hex())
		return nil, err
	}
	return hash, nil
}

func SwapBNBForExactTokens(ea *utils.ExtAcc, nonce uint64, index int) (*common.Hash, error) {
	var hash *common.Hash
	var err error
	// 50% wbnb will be returned
	actualAmount := new(big.Int)
	actualAmount.Div(utils.T_cfg.LiquidityTestAmount, big.NewInt(2))
	path := make([]common.Address, 0, 2)
	r := rand.Intn(10000) % 2
	if r == 0 {
		path = append(path, utils.T_cfg.Bep20AddrsA[index])
		path = append(path, utils.T_cfg.WbnbAddr)
		hash, err = ea.SwapExactTokensForBNB(nonce, utils.T_cfg.LiquidityTestAmount, path, ea.Addr)
		if err != nil {
			log.Println("error: SwapExactTokensForBNB:", err, path[0].Hex(), path[1].Hex())
			return nil, err
		}
	}
	if r == 1 {
		path = append(path, utils.T_cfg.WbnbAddr)
		path = append(path, utils.T_cfg.Bep20AddrsA[index])
		hash, err = ea.SwapBNBForExactTokens(nonce, utils.T_cfg.LiquidityTestAmount, actualAmount, path, ea.Addr)
		if err != nil {
			log.Println("error: SwapBNBForExactTokens:", err, path[0].Hex(), path[1].Hex())
			return nil, err
		}
	}
	return hash, nil
}

func ERC721MintOrTransfer(ea *utils.ExtAcc, nonce uint64, randomAddress *utils.ExtAcc) (*common.Hash, error) {
	var hash *common.Hash
	var err error
	subScenario := utils.RandScenario(utils.T_cfg.ERC721MintOrTransfer)
	if subScenario.Name == utils.ERC721Mint {
		hash, err = ea.MintERC721(nonce)
		if err != nil {
			log.Println("error: erc721Mint:", err)
			return nil, err
		}
	} else {
		tokenID, err := ea.GetOneERC721TokenID()
		if err != nil {
			log.Println("error: get erc721 tokenID:", err)
			hash, err = ea.MintERC721(nonce)
			if err != nil {
				log.Println("error: erc721Mint:", err)
				return nil, err
			}
		} else {
			_, err = ea.ApproveERC721(nonce, randomAddress.Addr, tokenID)
			if err != nil {
				log.Println("error: approve erc721: ", err, randomAddress.Addr.String())
				return nil, err
			}
			nonce++
			hash, err = ea.TransferERC721(nonce, randomAddress.Addr, tokenID)
			if err != nil {
				log.Println("error: transfer erc721: ", err)
				return nil, err
			}
		}
	}
	return hash, nil
}

func ERC1155MintOrBurnOrTransfer(ea *utils.ExtAcc, nonce uint64, randomAddress *utils.ExtAcc) (*common.Hash, error) {
	var hash *common.Hash
	var err error
	switch utils.RandScenario(utils.T_cfg.ERC1155MintOrBurnOrTransfer).Name {
	case utils.ERC1155Mint:
		randomTokenID := rand.Int63n(utils.T_cfg.Erc1155InitTokenTypeNumber)
		hash, err = ea.MintERC1155(nonce, big.NewInt(randomTokenID), big.NewInt(utils.T_cfg.Erc1155InitTokenNumber))
		if err != nil {
			log.Println("error: erc1155 Mint:", err)
			return nil, err
		}
	case utils.ERC1155Burn:
		id, err := ea.GetOneERC1155TokenID(utils.T_cfg.Erc1155TokenIDSlice)
		if err != nil {
			log.Println("error: get erc1155 tokenID:", err)
			randomTokenID := rand.Int63n(utils.T_cfg.Erc1155InitTokenTypeNumber)
			hash, err = ea.MintERC1155(nonce, big.NewInt(randomTokenID), big.NewInt(utils.T_cfg.Erc1155InitTokenNumber))
			if err != nil {
				log.Println("error: erc1155 Mint:", err)
				return nil, err
			}
		} else {
			hash, err = ea.BurnERC1155(nonce, id, big.NewInt(utils.T_cfg.Erc1155InitTokenNumber))
			if err != nil {
				log.Println("error: erc1155 Burn:", err)
				return nil, err
			}
		}
	case utils.ERC1155Transfer:
		id, err := ea.GetOneERC1155TokenID(utils.T_cfg.Erc1155TokenIDSlice)
		if err != nil {
			log.Println("error: get erc1155 tokenID:", err)
			randomTokenID := rand.Int63n(utils.T_cfg.Erc1155InitTokenTypeNumber)
			hash, err = ea.MintERC1155(nonce, big.NewInt(randomTokenID), big.NewInt(utils.T_cfg.Erc1155InitTokenNumber))
			if err != nil {
				log.Println("error: erc1155 Mint:", err)
				return nil, err
			}
		} else {
			hash, err = ea.TransERC1155(nonce, *randomAddress.Addr, id, big.NewInt(utils.T_cfg.Erc1155InitTokenNumber))
			if err != nil {
				log.Println("error: erc1155 Trans:", err)
				return nil, err
			}
		}
	}
	return hash, nil
}
