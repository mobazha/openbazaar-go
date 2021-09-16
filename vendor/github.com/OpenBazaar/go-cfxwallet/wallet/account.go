package wallet

import (
	"encoding/hex"

	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
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

// CfxAddress implements the WalletAddress interface
type CfxAddress struct {
	address *cfxaddress.Address
}

// New create conflux address by base32 string or hex40 string, if base32OrHex is base32 and networkID is passed it will create cfx Address use networkID.
func NewCfxAddress(base32OrHex string, networkID ...uint32) (*CfxAddress, error) {
	addr, err := cfxaddress.New(base32OrHex, networkID...)
	return &CfxAddress{&addr}, err
}

// String representation of cfx address
func (addr CfxAddress) String() string {
	return addr.address.String()
}

func (addr CfxAddress) VerboseString() string {
	return addr.address.MustGetVerboseBase32Address()
}

// EncodeAddress returns representation of the address
func (addr CfxAddress) EncodeAddress() string {
	return addr.String()
}

// ScriptAddress returns byte representation of address
func (addr CfxAddress) ScriptAddress() []byte {
	commonAddress := addr.address.MustGetCommonAddress()
	return commonAddress.Bytes()
}

// IsForNet returns true because CfxAddress has to become btc.Address
func (addr CfxAddress) IsForNet(params *chaincfg.Params) bool {
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
