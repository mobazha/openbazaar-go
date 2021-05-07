package migrations

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

type Migration035 struct{}

func (Migration035) Up(repoPath, dbPassword string, testnet bool) error {
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

	btc, ok := walletCfg["BTC"]
	if !ok {
		return errors.New("invalid config: missing BTC Wallet")
	}

	btcWalletCfg, ok := btc.(map[string]interface{})
	if !ok {
		return errors.New("invalid config: invalid BTC Wallet")
	}

	btcWalletCfg["API"] = []string{"https://btc1.mobazha.com/api"}
	btcWalletCfg["APITestnet"] = []string{"https://tbtc1.trezor.io/api"}

	bch, ok := walletCfg["BCH"]
	if !ok {
		return errors.New("invalid config: missing BCH Wallet")
	}

	bchWalletCfg, ok := bch.(map[string]interface{})
	if !ok {
		return errors.New("invalid config: invalid BCH Wallet")
	}

	bchWalletCfg["API"] = []string{"https://bch1.mobazha.com/api"}
	bchWalletCfg["APITestnet"] = []string{"https://tbch1.trezor.io/api"}

	ltc, ok := walletCfg["LTC"]
	if !ok {
		return errors.New("invalid config: missing LTC Wallet")
	}

	ltcWalletCfg, ok := ltc.(map[string]interface{})
	if !ok {
		return errors.New("invalid config: invalid LTC Wallet")
	}

	ltcWalletCfg["API"] = []string{"https://ltc1.mobazha.com/api"}
	ltcWalletCfg["APITestnet"] = []string{"https://tltc1.trezor.io/api"}

	zec, ok := walletCfg["ZEC"]
	if !ok {
		return errors.New("invalid config: missing ZEC Wallet")
	}

	zecWalletCfg, ok := zec.(map[string]interface{})
	if !ok {
		return errors.New("invalid config: invalid ZEC Wallet")
	}

	zecWalletCfg["API"] = []string{"https://zec1.mobazha.com/api"}
	zecWalletCfg["APITestnet"] = []string{"https://tzec1.trezor.io/api"}

	newConfigBytes, err := json.MarshalIndent(configMap, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal migrated config: %s", err.Error())
	}

	if err := ioutil.WriteFile(path.Join(repoPath, "config"), newConfigBytes, os.ModePerm); err != nil {
		return fmt.Errorf("writing migrated config: %s", err.Error())
	}

	if err := writeRepoVer(repoPath, 36); err != nil {
		return fmt.Errorf("bumping repover to 36: %s", err.Error())
	}
	return nil
}

func (Migration035) Down(repoPath, dbPassword string, testnet bool) error {
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

	btc, ok := walletCfg["BTC"]
	if !ok {
		return errors.New("invalid config: missing BTC Wallet")
	}

	btcWalletCfg, ok := btc.(map[string]interface{})
	if !ok {
		return errors.New("invalid config: invalid BTC Wallet")
	}

	btcWalletCfg["API"] = []string{"https://btc.trezor.io/api"}
	btcWalletCfg["APITestnet"] = []string{"https://tbtc.trezor.io/api"}

	bch, ok := walletCfg["BCH"]
	if !ok {
		return errors.New("invalid config: missing BCH Wallet")
	}

	bchWalletCfg, ok := bch.(map[string]interface{})
	if !ok {
		return errors.New("invalid config: invalid BCH Wallet")
	}

	bchWalletCfg["API"] = []string{"https://bch.trezor.io/api"}
	bchWalletCfg["APITestnet"] = []string{"https://tbch.trezor.io/api"}

	ltc, ok := walletCfg["LTC"]
	if !ok {
		return errors.New("invalid config: missing LTC Wallet")
	}

	ltcWalletCfg, ok := ltc.(map[string]interface{})
	if !ok {
		return errors.New("invalid config: invalid LTC Wallet")
	}

	ltcWalletCfg["API"] = []string{"https://ltc.trezor.io/api"}
	ltcWalletCfg["APITestnet"] = []string{"https://tltc.trezor.io/api"}

	zec, ok := walletCfg["ZEC"]
	if !ok {
		return errors.New("invalid config: missing ZEC Wallet")
	}

	zecWalletCfg, ok := zec.(map[string]interface{})
	if !ok {
		return errors.New("invalid config: invalid ZEC Wallet")
	}

	zecWalletCfg["API"] = []string{"https://zec.trezor.io/api"}
	zecWalletCfg["APITestnet"] = []string{"https://tzec.trezor.io/api"}

	newConfigBytes, err := json.MarshalIndent(configMap, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal migrated config: %s", err.Error())
	}

	if err := ioutil.WriteFile(path.Join(repoPath, "config"), newConfigBytes, os.ModePerm); err != nil {
		return fmt.Errorf("writing migrated config: %s", err.Error())
	}

	if err := writeRepoVer(repoPath, 35); err != nil {
		return fmt.Errorf("dropping repover to 35: %s", err.Error())
	}
	return nil
}
