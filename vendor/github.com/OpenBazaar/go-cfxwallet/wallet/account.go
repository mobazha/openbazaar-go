package wallet

import (
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	"github.com/btcsuite/btcd/chaincfg"
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
