package migrations

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/OpenBazaar/openbazaar-go/schema"
)

type migration036CoinConfig struct {
	APIPool            []string               `json:"API"`
	APITestnetPool     []string               `json:"APITestnet"`
	FeeAPI             string                 `json:"FeeAPI"`
	HighFeeDefault     uint64                 `json:"HighFeeDefault"`
	LowFeeDefault      uint64                 `json:"LowFeeDefault"`
	MaxFee             uint64                 `json:"MaxFee"`
	MediumFeeDefault   uint64                 `json:"MediumFeeDefault"`
	SuperLowFeeDefault uint64                 `json:"SuperLowFeeDefault"`
	TrustedPeer        string                 `json:"TrustedPeer"`
	Type               string                 `json:"Type"`
	WalletOptions      map[string]interface{} `json:"WalletOptions"`
}

var confluxDefaultConfig = migration036CoinConfig{
	Type:             schema.WalletTypeAPI,
	APIPool:          schema.CoinPoolCFX,
	APITestnetPool:   schema.CoinPoolTCFX,
	FeeAPI:           "", // intentionally blank
	LowFeeDefault:    7,
	MediumFeeDefault: 15,
	HighFeeDefault:   30,
	MaxFee:           200,
	WalletOptions:    schema.ConfluxDefaultOptions(),
}

type Migration036 struct{}

func (Migration036) Up(repoPath, dbPassword string, testnet bool) error {
	var (
		configMap        = map[string]interface{}{}
		configBytes, err = ioutil.ReadFile(path.Join(repoPath, "config"))
	)
	if err != nil {
		return fmt.Errorf("reading config: %s", err.Error())
	}

	if err = json.Unmarshal(configBytes, &configMap); err != nil {
		return fmt.Errorf("unmarshal config: %s", err.Error())
	}

	c, ok := configMap["Wallets"]
	if !ok {
		return errors.New("invalid config: missing key Wallets")
	}

	walletCfg, ok := c.(map[string]interface{})
	if !ok {
		return errors.New("invalid config: invalid key Wallets")
	}

	_, ok = walletCfg["CFX"]
	if ok {
		return errors.New("invalid config: already has CFX Wallet")
	}

	walletCfg["CFX"] = confluxDefaultConfig

	newConfigBytes, err := json.MarshalIndent(configMap, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal migrated config: %s", err.Error())
	}

	if err := ioutil.WriteFile(path.Join(repoPath, "config"), newConfigBytes, os.ModePerm); err != nil {
		return fmt.Errorf("writing migrated config: %s", err.Error())
	}

	if err := writeRepoVer(repoPath, 37); err != nil {
		return fmt.Errorf("bumping repover to 37: %s", err.Error())
	}
	return nil
}

func (Migration036) Down(repoPath, dbPassword string, testnet bool) error {
	var (
		configMap        = map[string]interface{}{}
		configBytes, err = ioutil.ReadFile(path.Join(repoPath, "config"))
	)
	if err != nil {
		return fmt.Errorf("reading config: %s", err.Error())
	}

	if err = json.Unmarshal(configBytes, &configMap); err != nil {
		return fmt.Errorf("unmarshal config: %s", err.Error())
	}

	c, ok := configMap["Wallets"]
	if !ok {
		return errors.New("invalid config: missing key Wallets")
	}

	walletCfg, ok := c.(map[string]interface{})
	if !ok {
		return errors.New("invalid config: invalid key Wallets")
	}

	_, ok = walletCfg["CFX"]
	if !ok {
		return errors.New("invalid config: missing CFX Wallet")
	}

	delete(walletCfg, "CFX")

	newConfigBytes, err := json.MarshalIndent(configMap, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal migrated config: %s", err.Error())
	}

	if err := ioutil.WriteFile(path.Join(repoPath, "config"), newConfigBytes, os.ModePerm); err != nil {
		return fmt.Errorf("writing migrated config: %s", err.Error())
	}

	if err := writeRepoVer(repoPath, 36); err != nil {
		return fmt.Errorf("dropping repover to 36: %s", err.Error())
	}
	return nil
}
