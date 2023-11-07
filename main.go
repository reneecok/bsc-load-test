package main

import (
	"bsc-load-test/log"
	"context"
	"encoding/hex"

	"bsc-load-test/contracts/V2factory"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// check init code hash, if deployment does not work properly
	client, _ := ethclient.Dial("http://172.22.42.160:8545")
	chainId, _ := client.ChainID(context.Background())
	networkId, _ := client.NetworkID(context.Background())
	log.Println("chainId:", chainId, "networkId:", networkId)
	txHash := common.HexToHash("0x9c546af09fa7ed33dd24571f3be0c69c2f049a25266cf7be9ccdfff96595810c")
	receipt, err := client.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		log.Errorf("error getting transaction receipt: %v", err)
		return
	}
	tipCap, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Errorf("error getting transaction receipt: %v", err)

		return
	}
	log.Infof("GasUsed: %d,tipCap: %s", receipt.GasUsed, tipCap.String())
	number, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Errorf("HeaderByNumber: %v", err)
		return
	}
	if number.BaseFee == nil {
		log.Errorf("BaseFee is nil: %v", number)

	}
	log.Infof("header by number: %v", number.BaseFee)
	v2factory, _ := V2factory.NewV2factory(common.HexToAddress("0xBb4991862A738837C131703dA2F29f7F3075A231"), client)
	initHash, _ := v2factory.GetPairInitHash(&bind.CallOpts{})
	log.Println(hex.EncodeToString(initHash[:]))
}
