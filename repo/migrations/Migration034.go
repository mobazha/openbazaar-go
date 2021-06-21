package migrations

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

var (
	Migration034BootstrapBefore = []string{
		"/ip4/107.170.133.32/tcp/4001/ipfs/QmUZRGLhcKXF1JyuaHgKm23LvqcoMYwtb9jmh8CkP4og3K",
		"/ip4/139.59.174.197/tcp/4001/ipfs/QmZfTbnpvPwxCjpCG3CXJ7pfexgkBZ2kgChAiRJrTK1HsM",
		"/ip4/139.59.6.222/tcp/4001/ipfs/QmRDcEDK9gSViAevCHiE6ghkaBCU7rTuQj4BDpmCzRvRYg",
	}

	Migration034BootstrapAfter = []string{
		"/ip4/45.76.183.141/tcp/4001/ipfs/QmV2B7fcVR6o8ZKs7D8vexhhQjjKZtofJzoFsx44X2ioEE",
		"/ip4/137.220.50.87/tcp/4001/ipfs/QmSqRoRDqGWd9VLQVAWHqLmBH6RW93CPY7vdqXCZELCt52",
		"/ip4/139.9.196.33/tcp/4001/ipfs/QmNPNz8nrpy5CfJiof7sv9XbPBvpxe3myP3HKcMF3WGofo",
		"/ip4/101.34.13.199/tcp/4001/ipfs/QmPeyynV8haCtFFfVhFRCiZopBU5EqET3opW6P8JwhSD5t",
	}

	Migration034PushToBefore = []string{
		"QmbwN82MVyBukT7WTdaQDppaACo62oUfma8dUa5R9nBFHm",
		"QmPPg2qeF3n2KvTRXRZLaTwHCw8JxzF4uZK93RfMoDvf2o",
		"QmY8puEnVx66uEet64gAf4VZRo7oUyMCwG6KdB9KM92EGQ",
	}

	Migration034PushToAfter = []string{
		"QmV2B7fcVR6o8ZKs7D8vexhhQjjKZtofJzoFsx44X2ioEE",
		"QmSqRoRDqGWd9VLQVAWHqLmBH6RW93CPY7vdqXCZELCt52",
		"QmNPNz8nrpy5CfJiof7sv9XbPBvpxe3myP3HKcMF3WGofo",
		"QmPeyynV8haCtFFfVhFRCiZopBU5EqET3opW6P8JwhSD5t",
	}
)

type migration034Bootstrap struct {
	PushTo []string
}

type migration034DataSharing struct {
	AcceptStoreRequests bool
	PushTo              []string
}

type Migration034 struct{}

func (Migration034) Up(repoPath, dbPassword string, testnet bool) error {
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

	configMap["DataSharing"] = migration034DataSharing{PushTo: Migration034PushToAfter}
	configMap["Bootstrap"] = Migration034BootstrapAfter

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

	btcWalletCfg["API"] = []string{"https://btc1.trezor.io/api"}
	btcWalletCfg["APITestnet"] = []string{"https://tbtc1.trezor.io/api"}

	bch, ok := walletCfg["BCH"]
	if !ok {
		return errors.New("invalid config: missing BCH Wallet")
	}

	bchWalletCfg, ok := bch.(map[string]interface{})
	if !ok {
		return errors.New("invalid config: invalid BCH Wallet")
	}

	bchWalletCfg["API"] = []string{"https://bch1.trezor.io/api"}
	bchWalletCfg["APITestnet"] = []string{"https://tbch1.trezor.io/api"}

	ltc, ok := walletCfg["LTC"]
	if !ok {
		return errors.New("invalid config: missing LTC Wallet")
	}

	ltcWalletCfg, ok := ltc.(map[string]interface{})
	if !ok {
		return errors.New("invalid config: invalid LTC Wallet")
	}

	ltcWalletCfg["API"] = []string{"https://ltc1.trezor.io/api"}
	ltcWalletCfg["APITestnet"] = []string{"https://tltc1.trezor.io/api"}

	zec, ok := walletCfg["ZEC"]
	if !ok {
		return errors.New("invalid config: missing ZEC Wallet")
	}

	zecWalletCfg, ok := zec.(map[string]interface{})
	if !ok {
		return errors.New("invalid config: invalid ZEC Wallet")
	}

	zecWalletCfg["API"] = []string{"https://zec1.trezor.io/api"}
	zecWalletCfg["APITestnet"] = []string{"https://tzec1.trezor.io/api"}

	newConfigBytes, err := json.MarshalIndent(configMap, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal migrated config: %s", err.Error())
	}

	if err := ioutil.WriteFile(path.Join(repoPath, "config"), newConfigBytes, os.ModePerm); err != nil {
		return fmt.Errorf("writing migrated config: %s", err.Error())
	}

	if err := writeRepoVer(repoPath, 35); err != nil {
		return fmt.Errorf("bumping repover to 35: %s", err.Error())
	}
	return nil
}

func (Migration034) Down(repoPath, dbPassword string, testnet bool) error {
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

	configMap["DataSharing"] = migration034DataSharing{PushTo: Migration034PushToBefore}
	configMap["Bootstrap"] = Migration034BootstrapBefore

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

	btcWalletCfg["API"] = []string{"https://btc.api.openbazaar.org/api"}
	btcWalletCfg["APITestnet"] = []string{"https://tbtc.api.openbazaar.org/api"}

	bch, ok := walletCfg["BCH"]
	if !ok {
		return errors.New("invalid config: missing BCH Wallet")
	}

	bchWalletCfg, ok := bch.(map[string]interface{})
	if !ok {
		return errors.New("invalid config: invalid BCH Wallet")
	}

	bchWalletCfg["API"] = []string{"https://bch.api.openbazaar.org/api"}
	bchWalletCfg["APITestnet"] = []string{"https://tbch.api.openbazaar.org/api"}

	ltc, ok := walletCfg["LTC"]
	if !ok {
		return errors.New("invalid config: missing LTC Wallet")
	}

	ltcWalletCfg, ok := ltc.(map[string]interface{})
	if !ok {
		return errors.New("invalid config: invalid LTC Wallet")
	}

	ltcWalletCfg["API"] = []string{"https://ltc.api.openbazaar.org/api"}
	ltcWalletCfg["APITestnet"] = []string{"https://tltc.api.openbazaar.org/api"}

	zec, ok := walletCfg["ZEC"]
	if !ok {
		return errors.New("invalid config: missing ZEC Wallet")
	}

	zecWalletCfg, ok := zec.(map[string]interface{})
	if !ok {
		return errors.New("invalid config: invalid ZEC Wallet")
	}

	zecWalletCfg["API"] = []string{"https://zec.api.openbazaar.org/api"}
	zecWalletCfg["APITestnet"] = []string{"https://tzec.api.openbazaar.org/api"}

	newConfigBytes, err := json.MarshalIndent(configMap, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal migrated config: %s", err.Error())
	}

	if err := ioutil.WriteFile(path.Join(repoPath, "config"), newConfigBytes, os.ModePerm); err != nil {
		return fmt.Errorf("writing migrated config: %s", err.Error())
	}

	if err := writeRepoVer(repoPath, 34); err != nil {
		return fmt.Errorf("dropping repover to 34: %s", err.Error())
	}
	return nil
}
