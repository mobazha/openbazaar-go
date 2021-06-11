package wallet

import (
	"errors"
	"fmt"
	"sync"
	"time"

	cfxSDK "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	wi "github.com/OpenBazaar/wallet-interface"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// CfxClient represents the cfx client
type CfxClient struct {
	*cfxSDK.Client
	url string
}

var txns []wi.Txn
var txnsLock sync.RWMutex

// NewCfxClient returns a new cfx client
func NewCfxClient(url string, option ...cfxSDK.ClientOption) (*CfxClient, error) {
	var conn *cfxSDK.Client

	var err error
	if conn, err = cfxSDK.NewClient(url, option...); err != nil {
		return nil, err
	}

	return &CfxClient{
		Client: conn,
		url:    url,
	}, nil

}

// Transfer will transfer cfx from this user account to dest address
func (client *CfxClient) Transfer(from cfxaddress.Address, to cfxaddress.Address, value *hexutil.Big, spendAll bool, fee *hexutil.Big) (types.Hash, error) {
	utx, err := client.CreateUnsignedTransaction(from, to, value, nil)
	if err != nil {
		return "", err
	}
	fmt.Printf("creat a new unsigned transaction %+v\n\n", utx)

	err = client.ApplyUnsignedTransactionDefault(&utx)
	if err != nil {
		return "", err
	}

	//sign
	if client.AccountManager == nil {
		return "", errors.New("account manager not specified, see SetAccountManager")
	}

	rawData, err := client.AccountManager.SignTransaction(utx)
	if err != nil {
		return "", err
	}

	//send raw tx
	txhash, err := client.SendRawTransaction(rawData)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction, raw data = 0x%+x, error: %v", rawData, err)
	}

	fmt.Printf("send transaction hash: %v\n\n", txhash)

	txns = append(txns, wi.Txn{
		Txid:      txhash.String(),
		Value:     value.String(),
		Height:    int32(utx.Nonce.ToInt().Int64()),
		Timestamp: time.Now(),
		WatchOnly: false,
		Bytes:     rawData})
	return txhash, nil
}

// EstimateTxnGas - returns estimated gas
func (client *CfxClient) EstimateTxnGas(from, to cfxaddress.Address, value *hexutil.Big) (*hexutil.Big, error) {
	utx, err := client.CreateUnsignedTransaction(from, to, value, nil)
	if err != nil {
		return nil, err
	}
	fmt.Printf("creat a new unsigned transaction %+v\n\n", utx)

	err = client.ApplyUnsignedTransactionDefault(&utx)
	if err != nil {
		return nil, err
	}

	return utx.Gas, nil
}

// EstimateGasSpend - returns estimated gas
func (client *CfxClient) EstimateGasSpend(from cfxaddress.Address, value *hexutil.Big) (*hexutil.Big, error) {
	// gas := big.NewInt(0)
	// gasPrice, err := client.SuggestGasPrice(context.Background())
	// if err != nil {
	// 	return gas, err
	// }
	// msg := ethereum.CallMsg{From: from, Value: value}
	// gasLimit, err := client.EstimateGas(context.Background(), msg)
	// if err != nil {
	// 	return gas, err
	// }
	// return gas.Mul(big.NewInt(int64(gasLimit)), gasPrice), nil
	return nil, nil
}

// GetTxnNonce - used to fetch nonce for a submitted txn
func (client *CfxClient) GetTxnNonce(txID string) (int32, error) {
	txnsLock.Lock()
	defer txnsLock.Unlock()
	for _, txn := range txns {
		if txn.Txid == txID {
			return txn.Height, nil
		}
	}
	return 0, errors.New("nonce not found")
}

type CfxGasStationData struct {
	Average     float64 `json:"average"`
	FastestWait float64 `json:"fastestWait"`
	FastWait    float64 `json:"fastWeight"`
	Fast        float64 `json:"Fast"`
	SafeLowWait float64 `json:"safeLowWait"`
	BlockNum    int64   `json:"blockNum"`
	AvgWait     float64 `json:"avgWait"`
	BlockTime   float64 `json:"block_time"`
	Speed       float64 `json:"speed"`
	Fastest     float64 `json:"fastest"`
	SafeLow     float64 `json:"safeLow"`
}

func (client *CfxClient) GetCfxGasStationEstimate() (*CfxGasStationData, error) {
	return &CfxGasStationData{}, nil
}

func init() {
	txns = []wi.Txn{}
}
