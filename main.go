package main

import (
	"context"
	"encoding/hex"
	"log"

	"bsc-load-test/contracts/V2factory"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// check init code hash, if deployment does not work properly
	client, _ := ethclient.Dial("http://172.22.41.197:8545")
	chainId, _ := client.ChainID(context.Background())
	networkId, _ := client.NetworkID(context.Background())
	log.Println("chainId:", chainId, "networkId:", networkId)
	v2factory, _ := V2factory.NewV2factory(common.HexToAddress("0xBb4991862A738837C131703dA2F29f7F3075A231"), client)
	initHash, _ := v2factory.GetPairInitHash(&bind.CallOpts{})
	log.Println(hex.EncodeToString(initHash[:]))
}
