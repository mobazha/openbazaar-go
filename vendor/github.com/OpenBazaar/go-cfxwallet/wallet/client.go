package wallet

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	cfxSDK "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/constants"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	wi "github.com/OpenBazaar/wallet-interface"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// CfxClient represents the cfx client
type CfxClient struct {
	*cfxSDK.Client
	scaner *ConfluxScan
	url    string
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

	scaner := NewConfluxScan()

	return &CfxClient{
		Client: conn,
		scaner: scaner,
		url:    url,
	}, nil
}

func (client *CfxClient) GetFeePerByte(feeLevel wi.FeeLevel) big.Int {
	// est, err := wallet.client.GetCfxGasStationEstimate()
	// ret := big.NewInt(0)
	// if err != nil {
	// 	log.Errorf("err fetching ethgas station data: %v", err)
	// 	return *ret
	// }
	// switch feeLevel {
	// case wi.NORMAL:
	// 	ret, _ = big.NewFloat(est.Average * 100000000).Int(nil)
	// case wi.ECONOMIC, wi.SUPER_ECONOMIC:
	// 	ret, _ = big.NewFloat(est.SafeLow * 100000000).Int(nil)
	// case wi.PRIORITY, wi.FEE_BUMP:
	// 	ret, _ = big.NewFloat(est.Fast * 100000000).Int(nil)
	// }
	// return *ret

	gasPrice, err := client.GetGasPrice()
	if err != nil {
		gasPrice = types.NewBigInt(constants.MinGasprice)
	}

	// conflux responsed gasprice offen be 0, but the min gasprice is 1 when sending transaction, so do this
	if gasPrice.ToInt().Cmp(big.NewInt(constants.MinGasprice)) < 1 {
		gasPrice = types.NewBigInt(constants.MinGasprice)
	}
	tmp := hexutil.Big(*gasPrice)
	return *tmp.ToInt()
}

// GetBalance returns the balance for the wallet
func (client *CfxClient) getBalance() (*big.Int, error) {
	defaultAccount, _ := client.AccountManager.GetDefault()
	balance, err := client.GetBalance(*defaultAccount, types.EpochLatestConfirmed)
	return balance.ToInt(), err
}

// GetUnconfirmedBalance returns the unconfirmed balance for the wallet
func (client *CfxClient) getUnconfirmedBalance() (*big.Int, error) {
	defaultAccount, _ := client.AccountManager.GetDefault()
	balance, err := client.GetBalance(*defaultAccount, types.EpochLatestState)
	return balance.ToInt(), err
}

func (client *CfxClient) balanceCheckForContract(feeLevel wi.FeeLevel, amount big.Int) bool {
	fee := client.GetFeePerByte(feeLevel)
	if fee.Int64() == 0 {
		return false
	}
	// lets check if the caller has enough balance to make the
	// multisign call
	requiredBalance := new(big.Int).Mul(&fee, big.NewInt(maxGasLimit))
	requiredBalance = new(big.Int).Add(requiredBalance, &amount)

	currentBalance, err := client.getBalance()
	if err != nil {
		log.Error("err fetching cfx wallet balance")
		currentBalance = big.NewInt(0)
	}
	if requiredBalance.Cmp(currentBalance) > 0 {
		// the wallet does not have the required balance
		return false
	}
	return true
}

func (client *CfxClient) balanceCheck(feeLevel wi.FeeLevel, amount big.Int) bool {
	gasPrice := client.GetFeePerByte(feeLevel)
	if gasPrice.Int64() == 0 {
		return false
	}

	from, _ := client.AccountManager.GetDefault()
	gasVal := hexutil.Big(amount)
	gasLimit, err := client.EstimateGasSpend(*from, &gasVal)
	if err != nil {
		return false
	}

	currentBalance, err := client.getBalance()
	if err != nil {
		//currentBalance = big.NewInt(0)
		return false
	}
	gas := new(big.Int).Mul(&gasPrice, &gasLimit)

	requiredBalance := new(big.Int).Add(&amount, gas)
	return currentBalance.Cmp(requiredBalance) > 0
}

// Transfer will transfer cfx from this user account to dest address
func (client *CfxClient) Transfer(from cfxaddress.Address, to cfxaddress.Address, value *hexutil.Big, spendAll bool, fee *hexutil.Big) (types.Hash, error) {
	gasPrice := client.GetFeePerByte(wi.NORMAL)

	if gasPrice.Int64() < fee.ToInt().Int64() {
		gasPrice = *fee.ToInt()
	}

	tvalue := value

	gasLimit, err := client.EstimateGasSpend(from, value)
	if err != nil {
		return "", err
	}

	// if spend all then we need to set the value = confirmedBalance - gas
	if spendAll {
		currentBalance, err := client.getBalance()
		if err != nil {
			//currentBalance = big.NewInt(0)
			return "", err
		}
		gas := new(big.Int).Mul(&gasPrice, &gasLimit)

		if currentBalance.Cmp(gas) >= 0 {
			val := new(big.Int).Sub(currentBalance, gas)
			valTmp := hexutil.Big(*val)
			tvalue = &valTmp
		}
	}

	utx, err := client.CreateUnsignedTransaction(from, to, tvalue, nil)
	if err != nil {
		return "", err
	}

	err = client.ApplyUnsignedTransactionDefault(&utx)
	if err != nil {
		return "", err
	}

	//sign
	if client.AccountManager == nil {
		return "", errors.New("account manager not specified, see SetAccountManager")
	}

	err = client.AccountManager.TimedUnlockDefault("", 30*time.Second)
	if err != nil {
		return "", errors.New("account manager failed to unlock default address")
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

	txns = append(txns, wi.Txn{
		Txid:      txhash.String(),
		Value:     tvalue.String(),
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

	err = client.ApplyUnsignedTransactionDefault(&utx)
	if err != nil {
		return nil, err
	}

	return utx.Gas, nil
}

// EstimateGasSpend - returns estimated gas
func (client *CfxClient) EstimateGasSpend(from cfxaddress.Address, value *hexutil.Big) (big.Int, error) {
	gas := big.NewInt(0)
	gasPrice, err := client.GetGasPrice()
	if err != nil {
		return *gas, err
	}

	to := "cfxtest:aap8z8k87yptj485j96amtskgur201btpuu54e76wp"
	networkID, _ := client.GetNetworkID()
	toAddr := cfxaddress.MustNew(to, networkID)

	gasLimit, err := client.EstimateTxnGas(from, toAddr, value)
	if err != nil {
		return *gas, err
	}
	return *gas.Mul(gasLimit.ToInt(), gasPrice.ToInt()), nil
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
