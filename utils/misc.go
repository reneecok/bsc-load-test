package utils

import (
	"bufio"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	SendBNB = "SendBNB"
	// for bep20 contract
	SendBEP20 = "SendBEP20"
	// for uniswap contract
	AddLiquidity = "AddLiquidity"
	RemoveLiquidity = "RemoveLiquidity"
	SwapExactTokensForTokens = "SwapExactTokensForTokens"
	SwapBNBForExactTokens = "SwapBNBForExactTokens"
	// for wbnb contract
	DepositWBNB = "DepositWBNB"
	WithdrawWBNB = "WithdrawWBNB"
)

type Scenario struct {
	Name    string
	Weight  int
}

func RandScenario(scenarios []Scenario) *Scenario {
	//
	var totalWeight int
	for _, v := range scenarios {
		totalWeight += v.Weight
	}
	//
	r := rand.Intn(totalWeight)
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
	items := make([][]string, 0, numOfKeys)
	keyfile, err := os.Open(fullpath)
	if err != nil {
		panic(err)
	}
	defer keyfile.Close()
	//
	index := 0
	scan := bufio.NewScanner(keyfile)
	for scan.Scan() {
		if index == numOfKeys {
			break
		}
		line := scan.Text()
		item := strings.Split(line, ",")
		items = append(items, item)
		index++
	}
	return items
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
