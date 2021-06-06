package wallet

import (
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	"github.com/btcsuite/btcd/chaincfg"
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
