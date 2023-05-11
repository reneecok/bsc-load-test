package utils

import (
	"bufio"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/ratelimit"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	SendBNB = "SendBNB"
	// for bep20 contract
	SendBEP20 = "SendBEP20"
	// for uniswap contract
	AddLiquidity             = "AddLiquidity"
	RemoveLiquidity          = "RemoveLiquidity"
	SwapExactTokensForTokens = "SwapExactTokensForTokens"
	SwapBNBForExactTokens    = "SwapBNBForExactTokens"
	// for wbnb contract
	DepositWBNB  = "DepositWBNB"
	WithdrawWBNB = "WithdrawWBNB"
	// for NFT contract
	ERC721MintOrTransfer        = "ERC721MintOrTransfer"
	ERC721Mint                  = "ERC721Mint"
	ERC721Transfer              = "ERC721Transfer"
	ERC1155MintOrBurnOrTransfer = "ERC1155MintOrBurnOrTransfer"
	ERC1155Mint                 = "ERC1155Mint"
	ERC1155Burn                 = "ERC1155Burn"
	ERC1155Transfer             = "ERC1155Transfer"
)

type Scenario struct {
	Name   string
	Weight int
}

func RandScenario(scenarios []Scenario) *Scenario {
	//
	var totalWeight int
	for _, v := range scenarios {
		totalWeight += v.Weight
	}
	//
	r := rand.Intn(totalWeight) + 1
	for _, v := range scenarios {
		r -= v.Weight
		if r <= 0 {
			return &v
		}
	}
	return nil
}

/* keyfile content: key,addr
1b50db67b99c97a60a8e04bce34c5e716067e6be6a4d9b6d300438a2090e086e,0xA41C4328af279D96d932372C22f30a837ebaA1f0
132933a040e54d42c865461318d5256a1112261ec296d3d653e3121a76988643,0x05A49BFC4c4A4597839342baF0178C8bFCB58c1D
......
*/

func RandHexKeys(fullpath string, numOfKeys int) {
	keyfile, err := os.OpenFile(fullpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer keyfile.Close()
	writer := bufio.NewWriter(keyfile)
	for i := 0; i < numOfKeys; i++ {
		key, err := crypto.GenerateKey()
		if err != nil {
			panic(err)
		}
		keyBytes := crypto.FromECDSA(key)
		// strip off the 0x after hex encoded
		hexkey := hexutil.Encode(keyBytes)[2:]
		pubKey := key.Public()
		pubKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
		if !ok {
			err = errors.New("publicKey is not *ecdsa.PublicKey")
			panic(err)
		}
		addr := crypto.PubkeyToAddress(*pubKeyECDSA)
		//
		line := fmt.Sprintf("%s,%s\n", hexkey, addr.Hex())
		_, err = writer.WriteString(line)
		if err != nil {
			panic(err.Error())
		}
	}
	err = writer.Flush()
	if err != nil {
		panic(err.Error())
	}
}

func LoadHexKeys(fullpath string, numOfKeys int) [][]string {
	var batches [][]string
	batchSize := 100000
	if numOfKeys <= batchSize {
		batches = make([][]string, 0, 1)
	} else {
		batches = make([][]string, 0, (numOfKeys/batchSize)+1)
	}
	keyfile, err := os.Open(fullpath)
	if err != nil {
		panic(err)
	}
	defer keyfile.Close()
	//
	index := 0
	lines := make([]string, 0, batchSize)
	scanner := bufio.NewScanner(keyfile)
	for scanner.Scan() {
		if index == numOfKeys {
			break
		}
		line := scanner.Text()
		if index != 0 && index%batchSize == 0 {
			batches = append(batches, lines)
			lines = make([]string, 0, batchSize)
		}
		lines = append(lines, line)
		index++
	}
	if len(lines) > 0 {
		batches = append(batches, lines)
	}
	return batches
}

func SaveHash(fullpath string, results []*common.Hash) error {
	keyfile, err := os.OpenFile(fullpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer keyfile.Close()
	writer := bufio.NewWriter(keyfile)
	for _, v := range results {
		line := v.Hex() + "\n"
		_, err = writer.WriteString(line)
		if err != nil {
			panic(err.Error())
		}
	}
	err = writer.Flush()
	if err != nil {
		panic(err.Error())
	}
	return nil
}

func Load(clients []*ethclient.Client, hexkeyfile string, usersLoaded *int) []ExtAcc {
	batches := LoadHexKeys(hexkeyfile, *usersLoaded)
	eaSlice := make([]ExtAcc, 0, *usersLoaded)
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
				ea, err := NewExtAcc(client, items[0], items[1])
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
func CheckAllTransactionStatus(root *ExtAcc, hashList []*common.Hash, tps int) {
	var wg sync.WaitGroup
	var numberLock sync.Mutex
	wg.Add(len(hashList))
	limiter := ratelimit.New(tps)
	txnFinishedNumber := 0
	for i := 0; i < len(hashList); i++ {
		limiter.Take()
		receipt := root.GetReceipt(hashList[i], 10)
		if receipt != nil && receipt.Status == 1 {
			numberLock.Lock()
			txnFinishedNumber++
			numberLock.Unlock()
		}
	}
	log.Println("tx hash returned in load test: ", len(hashList))
	log.Println("tx finished in load test: ", txnFinishedNumber)
}

func SetupTimer(dur time.Duration) *bool {
	t := time.NewTimer(dur)
	expired := false
	go func() {
		<-t.C
		expired = true
	}()
	return &expired
}
