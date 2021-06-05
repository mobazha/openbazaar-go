package wallet

import (
	"crypto/ecdsa"

	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
	"github.com/tyler-smith/go-bip39"
)

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
	return addr.String()
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

func GetPrivateKey(mnemonic string, password string) (string, error) {
	seed := bip39.NewSeed(mnemonic, "") //这里可以选择传入指定密码或者空字符串，不同密码生成的助记词不同

	wallet, err := hdwallet.NewFromSeed(seed)
	if err != nil {
		log.Fatal(err)
	}

	path := hdwallet.MustParseDerivationPath("m/44'/503'/0'/0/0") //最后一位是同一个助记词的地址id，从0开始，相同助记词可以生产无限个地址
	account, err := wallet.Derive(path, false)
	if err != nil {
		log.Fatal(err)
	}

	return wallet.PrivateKeyHex(account)
}

// Account represents ethereum keystore
type Account struct {
	// key *keystore.Key
	privateKey *ecdsa.PrivateKey
	address    common.Address
}

// NewAccountFromMnemonic returns generated account
func NewAccountFromMnemonic(mnemonic, password string) (*Account, error) {
	keyString, err := GetPrivateKey(mnemonic, password)
	if err != nil {
		return nil, err
	}

	privateKeyECDSA, err := crypto.HexToECDSA(keyString)
	if err != nil {
		return nil, err
	}

	return &Account{privateKey: privateKeyECDSA, address: crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)}, nil
}

// Address returns the cfx address
func (account *Account) Address() common.Address {
	return account.address
}
