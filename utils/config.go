package utils

import (
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"bsc-load-test/log"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Endpoints  string `yaml:"Endpoints"`
	Fullnodes  []string
	ChainIdYml int `yaml:"ChainId"`
	ChainId    *big.Int

	Roothexkey  string `yaml:"Roothexkey"`
	Roothexaddr string `yaml:"Roothexaddr"`

	SlaveUserHexkeyFile string `yaml:"SlaveUserHexkeyFile"`
	Hexkeyfile          string `yaml:"Hexkeyfile"`

	UsersCreated    int `yaml:"UsersCreated"`
	UsersLoaded     int `yaml:"UsersLoaded"`
	SlaveUserLoaded int `yaml:"SlaveUserLoaded"`

	Bep20Hex          string `yaml:"Bep20Hex"`
	WbnbHex           string `yaml:"WbnbHex"`
	UniswapFactoryHex string `yaml:"UniswapFactoryHex"`
	UniswapRouterHex  string `yaml:"UniswapRouterHex"`
	Erc721Hex         string `yaml:"Erc721Hex"`
	Erc1155Hex        string `yaml:"Erc1155Hex"`

	Bep20AddrsA        []common.Address
	Bep20AddrsB        []common.Address
	WbnbAddr           common.Address
	UniswapFactoryAddr common.Address
	UniswapRouterAddr  common.Address
	Erc721Addr         common.Address
	Erc1155Addr        common.Address

	Tps int `yaml:"Tps"`
	Sec int `yaml:"Sec"`

	ScenariosYml                            map[string]int `yaml:"ScenariosYml"`
	ERC721MintOrTransferScenariosYml        map[string]int `yaml:"ERC721MintOrTransferScenariosYml"`
	ERC1155MintOrBurnOrTransferScenariosYml map[string]int `yaml:"ERC1155MintOrBurnOrTransferScenariosYml"`
	Scenarios                               []Scenario
	ERC721MintOrTransfer                    []Scenario
	ERC1155MintOrBurnOrTransfer             []Scenario

	DistributeAmountYml        float64 `yaml:"DistributeAmountYml"`
	Erc721InitTokenNumber      int64   `yaml:"Erc721InitTokenNumber"`
	Erc1155InitTokenTypeNumber int64   `yaml:"Erc1155InitTokenTypeNumber"`
	Erc1155InitTokenNumber     int64   `yaml:"Erc1155InitTokenNumber"`

	DistributeAmount      *big.Int
	LiquidityInitAmount   *big.Int
	LiquidityTestAmount   *big.Int
	SlaveDistributeAmount *big.Int
	Erc1155TokenIDSlice   []*big.Int
}

var T_cfg = &Config{}

func (cfg *Config) LoadYml(tps, sec *int) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(wd, "config.yml")
	//
	configYML, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(configYML, cfg)
	if err != nil {
		return err
	}

	if *tps != -10 {
		cfg.Tps = *tps
	}

	if *sec != -10 {
		cfg.Sec = *sec
	}

	cfg.Fullnodes = strings.Split(cfg.Endpoints, ",")
	cfg.ChainId = big.NewInt(int64(cfg.ChainIdYml))
	tokens := strings.Split(cfg.Bep20Hex, ",")

	for _, v := range tokens[0 : len(tokens)/2] {
		cfg.Bep20AddrsA = append(cfg.Bep20AddrsA, common.HexToAddress(v))
	}
	for _, v := range tokens[len(tokens)/2:] {
		cfg.Bep20AddrsB = append(cfg.Bep20AddrsB, common.HexToAddress(v))
	}
	if cfg.Bep20Hex != "" && len(cfg.Bep20AddrsA) != len(cfg.Bep20AddrsB) {
		panic("unbalanced bep20 pair(s) found")
	}

	cfg.WbnbAddr = common.HexToAddress(cfg.WbnbHex)
	cfg.UniswapFactoryAddr = common.HexToAddress(cfg.UniswapFactoryHex)
	cfg.UniswapRouterAddr = common.HexToAddress(cfg.UniswapRouterHex)
	cfg.Erc721Addr = common.HexToAddress(cfg.Erc721Hex)
	cfg.Erc1155Addr = common.HexToAddress(cfg.Erc1155Hex)

	for i := 0; i < int(cfg.Erc1155InitTokenTypeNumber); i++ {
		cfg.Erc1155TokenIDSlice = append(cfg.Erc1155TokenIDSlice, big.NewInt(int64(i)))
	}

	distributeAmount := big.NewFloat(1e18).Mul(big.NewFloat(cfg.DistributeAmountYml), big.NewFloat(1e18))

	cfg.DistributeAmount, _ = distributeAmount.Int(cfg.DistributeAmount)
	log.Println(cfg.DistributeAmount.Int64())

	cfg.LiquidityInitAmount = new(big.Int)
	cfg.LiquidityTestAmount = new(big.Int)

	cfg.LiquidityInitAmount.Div(cfg.DistributeAmount, big.NewInt(4))
	cfg.LiquidityTestAmount.Div(cfg.LiquidityInitAmount, big.NewInt(2.5e12))
	log.Infof("===LiquidityInitAmount: %d, LiquidityTestAmount: %d===", cfg.LiquidityInitAmount, cfg.LiquidityTestAmount)
	copyAmount := big.NewInt(cfg.DistributeAmount.Int64())
	copyAmount.Mul(copyAmount, big.NewInt(int64(cfg.UsersLoaded)+int64(cfg.UsersLoaded/100)))
	copyAmount.Div(copyAmount, big.NewInt(int64(cfg.SlaveUserLoaded)))
	cfg.SlaveDistributeAmount = copyAmount

	for k, v := range cfg.ScenariosYml {
		cfg.Scenarios = append(cfg.Scenarios, Scenario{k, v})
	}

	for k, v := range cfg.ERC721MintOrTransferScenariosYml {
		cfg.ERC721MintOrTransfer = append(cfg.ERC721MintOrTransfer, Scenario{k, v})
	}

	for k, v := range cfg.ERC1155MintOrBurnOrTransferScenariosYml {
		cfg.ERC1155MintOrBurnOrTransfer = append(cfg.ERC1155MintOrBurnOrTransfer, Scenario{k, v})
	}

	log.Printf("Init config success! Endpoint: %s, tps: %d, sec: %d, userLoaded: %d, slaveUserLoaded: %d, distributeAmount: %d, slaveDistributeAmount: %d",
		cfg.Endpoints, cfg.Tps, cfg.Sec, cfg.UsersLoaded, cfg.SlaveUserLoaded, cfg.DistributeAmount.Int64(), cfg.SlaveDistributeAmount.Int64())

	return nil
}
