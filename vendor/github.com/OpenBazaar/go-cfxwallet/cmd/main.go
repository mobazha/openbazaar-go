package main

import (
	"github.com/OpenBazaar/go-cfxwallet/wallet"

	"github.com/OpenBazaar/multiwallet/config"
	"github.com/prometheus/common/log"
)

func main() {
	cfg := config.CoinConfig{}

	var mnemonic = "thank buddy dolphin gesture one tree doctor town uphold song nasty hint"
	wallet, err := wallet.NewConfluxWallet(cfg, mnemonic, ".", nil)
	if err != nil {
		log.Errorf("error initializing wallet: %s", err.Error())
		return
	}

	balance, err := wallet.GetBalance()
	if err != nil {
		log.Errorf("Faild to get balance: %s", err.Error())
		return
	}
	log.Infof("The balance is: %v", balance.String())
}
