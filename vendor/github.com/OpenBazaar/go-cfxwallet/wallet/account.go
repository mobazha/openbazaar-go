package wallet

import (
	"encoding/hex"

	"github.com/btcsuite/btcd/chaincfg"
	hd "github.com/btcsuite/btcutil/hdkeychain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tyler-smith/go-bip39"
)

// EthAddress implements the WalletAddress interface
type EthAddress struct {
	address *common.Address
}

// String representation of eth address
func (addr EthAddress) String() string {
	return addr.address.Hex() // [2:] //String()[2:]
}

// EncodeAddress returns hex representation of the address
func (addr EthAddress) EncodeAddress() string {
	return addr.address.Hex() // [2:]
}

// ScriptAddress returns byte representation of address
func (addr EthAddress) ScriptAddress() []byte {
	return addr.address.Bytes()
}

// IsForNet returns true because EthAddress has to become btc.Address
func (addr EthAddress) IsForNet(params *chaincfg.Params) bool {
	return true
}

func GetPrivateKey(mnemonic string, password string, params *chaincfg.Params) (string, error) {
	seed := bip39.NewSeed(mnemonic, password)

	privKey, err := hd.NewMaster(seed, params)
	if err != nil {
		log.Errorf("err initializing btc priv key : %v", err)
		return "", err
	}

	exPrivKey, err := privKey.ECPrivKey()
	if err != nil {
		log.Errorf("err extracting btcec priv key : %v", err)
		return "", err
	}

	privateKeyECDSA := exPrivKey.ToECDSA()

	keyString := hex.EncodeToString(crypto.FromECDSA(privateKeyECDSA))
	return keyString, nil
}
