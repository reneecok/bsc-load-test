package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"time"

	"bsc-load-test/executor"
	"bsc-load-test/utils"

	"github.com/ethereum/go-ethereum/ethclient"
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
	root, err := utils.NewExtAcc(clients[0], utils.T_cfg.Roothexkey, utils.T_cfg.Roothexaddr)
	if err != nil {
		panic(err)
	}
	log.Println("root:", root.Addr.Hex())
	nonce, err := root.Client.PendingNonceAt(context.Background(), *root.Addr)
	if err != nil {
		panic(err)
	}
	log.Println("root: nonce -", nonce)

	preChecker(err, root)

	if *initTestAcc {
		executor.InitAccount(clients, nonce, *root)
		return
	}
	//
	if *resetTestAcc {
		executor.ResetTest(clients, nonce, root)
		return
	}
	//
	if *runTestAcc {
		executor.Run(clients, root)
		return
	}
	//
	if *queryBlocks {
		root.GetBlockTrans(*blockNumS, *blockNumE)
		return
	}
}

func preChecker(err error, root *utils.ExtAcc) {
	if _, err = root.GetBNBBalance(); err != nil {
		panic(err)
	}
	if utils.T_cfg.Bep20Hex != "" {
		for i := range utils.T_cfg.Bep20AddrsA {
			if _, err = root.GetBEP20Balance(&utils.T_cfg.Bep20AddrsA[i]); err != nil {
				panic(err)
			}
			if _, err = root.GetBEP20Balance(&utils.T_cfg.Bep20AddrsB[i]); err != nil {
				panic(err)
			}
		}
	}
	if utils.T_cfg.WbnbHex != "" {
		if _, err = root.GetBEP20Balance(&utils.T_cfg.WbnbAddr); err != nil {
			panic(err)
		}
	}
}

func cleanup(clients []*ethclient.Client) {
	for _, v := range clients {
		v.Close()
	}
}
