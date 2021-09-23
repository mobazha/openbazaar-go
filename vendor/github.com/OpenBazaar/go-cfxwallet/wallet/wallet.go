package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"path"
	"sort"
	"strings"
	"time"

	"cfxabigen/bind"

	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	"github.com/Conflux-Chain/go-conflux-sdk/utils"
	"github.com/OpenBazaar/go-ethwallet/util"
	mwConfig "github.com/OpenBazaar/multiwallet/config"
	ut "github.com/OpenBazaar/openbazaar-go/util"
	wi "github.com/OpenBazaar/wallet-interface"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	hd "github.com/btcsuite/btcutil/hdkeychain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/op/go-logging"
	"golang.org/x/net/proxy"
)

// var _ = wi.Wallet(&ConfluxWallet{})
var done, doneBalanceTicker chan bool

const (
	maxGasLimit = 400000
)

var (
	emptyChainHash *chainhash.Hash

	// CfxCurrencyDefinition is cfx defaults
	CfxCurrencyDefinition = wi.CurrencyDefinition{
		Code:         "CFX",
		Divisibility: 18,
	}
	log = logging.MustGetLogger("cfxwallet")
)

func init() {
	mustInitEmptyChainHash()
}

func mustInitEmptyChainHash() {
	hash, err := chainhash.NewHashFromStr("")
	if err != nil {
		panic(fmt.Sprintf("creating emptyChainHash: %s", err.Error()))
	}
	emptyChainHash = hash
}

// CfxConfiguration - used for cfx specific configuration
type CfxConfiguration struct {
}

// CfxRedeemScript - used to represent redeem script for cfx wallet
// <uniqueId: 20><threshold:1><timeoutHours:4><buyer:20><seller:20>
// <moderator:20><multisigAddress:20><tokenAddress:20>
type CfxRedeemScript struct {
	TxnID           common.Address
	Threshold       uint8
	Timeout         uint32
	Buyer           common.Address
	Seller          common.Address
	Moderator       common.Address
	MultisigAddress common.Address
	TokenAddress    common.Address
}

// SerializeCfxScript - used to serialize cfx redeem script
func SerializeCfxScript(scrpt CfxRedeemScript) ([]byte, error) {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(scrpt)
	return b.Bytes(), err
}

// DeserializeCfxScript - used to deserialize cfx redeem script
func DeserializeCfxScript(b []byte) (CfxRedeemScript, error) {
	scrpt := CfxRedeemScript{}
	buf := bytes.NewBuffer(b)
	d := gob.NewDecoder(buf)
	err := d.Decode(&scrpt)
	return scrpt, err
}

// PendingTxn used to record a pending cfx txn
type PendingTxn struct {
	TxnID     types.Hash
	OrderID   string
	Amount    string
	Nonce     int32
	From      string
	To        string
	WithInput bool
}

// SerializePendingTxn - used to serialize cfx pending txn
func SerializePendingTxn(pTxn PendingTxn) ([]byte, error) {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(pTxn)
	return b.Bytes(), err
}

// DeserializePendingTxn - used to deserialize cfx pending txn
func DeserializePendingTxn(b []byte) (PendingTxn, error) {
	pTxn := PendingTxn{}
	buf := bytes.NewBuffer(b)
	d := gob.NewDecoder(buf)
	err := d.Decode(&pTxn)
	return pTxn, err
}

// GenScriptHash - used to generate script hash for cfx as per
// escrow smart contract
func GenScriptHash(script CfxRedeemScript) ([32]byte, string, error) {
	a := make([]byte, 4)
	binary.BigEndian.PutUint32(a, script.Timeout)
	arr := append(script.TxnID.Bytes(), append([]byte{script.Threshold},
		append(a[:], append(script.Buyer.Bytes(),
			append(script.Seller.Bytes(), append(script.Moderator.Bytes(),
				append(script.MultisigAddress.Bytes())...)...)...)...)...)...)
	var retHash [32]byte

	copy(retHash[:], crypto.Keccak256(arr)[:])
	ahashStr := hexutil.Encode(retHash[:])

	return retHash, ahashStr, nil
}

// ConfluxWallet is the wallet implementation for conflux
type ConfluxWallet struct {
	cfg     mwConfig.CoinConfig
	client  *CfxClient
	am      *sdk.AccountManager
	address *cfxaddress.Address

	registry *Registry
	ppsct    *Escrow

	db            wi.Datastore
	exchangeRates wi.ExchangeRates
	listeners     []func(wi.TransactionCallback)
}

// NewConfluxWallet will return a reference to the Cfx Wallet
func NewConfluxWallet(cfg mwConfig.CoinConfig, params *chaincfg.Params, mnemonic string, repoPath string, proxy proxy.Dialer) (*ConfluxWallet, error) {
	url := "https://test.confluxrpc.com"

	var networkID uint32 = 1029
	if cfg.CoinType == wi.TestnetConflux {
		networkID = 1
	}

	keyStorePath := path.Join(repoPath, "keystore")
	am := sdk.NewAccountManager(keyStorePath, networkID)
	_, err := am.GetDefault()
	if err != nil {
		log.Errorf("No account found: %s", err.Error())

		privateKey, err := GetPrivateKey(mnemonic, "", params)
		if err != nil {
			log.Errorf("get private key from mnemonic failed: %s", err.Error())
			return nil, err
		}

		_, err = am.ImportKey(privateKey, "")
		if err != nil {
			log.Errorf("import key failed: %s", err.Error())
			return nil, err
		}
	}

	client, err := NewCfxClient(url, sdk.ClientOption{KeystorePath: keyStorePath})
	if err != nil {
		log.Errorf("error initializing wallet: %s", err.Error())
		return nil, err
	}
	address, err := client.AccountManager.GetDefault()
	if err != nil {
		log.Errorf("get default failed: %s", err.Error())
		return nil, err
	}
	log.Infof("The address is: %v", address.String())

	var regAddr interface{}
	var ok bool
	registryKey := "TestnetRegistryAddress"
	if strings.Contains(cfg.ClientAPIs[0], "testnet") {
		registryKey = "TestnetRegistryAddress"
	}
	if regAddr, ok = cfg.Options[registryKey]; !ok {
		log.Errorf("conflux registry not found: %s", cfg.Options[registryKey])
		return nil, err
	}

	registryAddress, err := cfxaddress.NewFromBase32(regAddr.(string))
	if err != nil {
		log.Fatalf("error get contract adress: %s", err.Error())
	}

	reg, err := NewRegistry(registryAddress, client)
	if err != nil {
		log.Fatalf("error initilaizing contract failed: %s", err.Error())
	}

	er := NewConfluxPriceFetcher(proxy)

	return &ConfluxWallet{cfg, client, am, address, reg, nil, cfg.DB, er, []func(wi.TransactionCallback){}}, nil
}

// Params - return nil to comply
func (wallet *ConfluxWallet) Params() *chaincfg.Params {
	return nil
}

// GetBalance returns the balance for the wallet
func (wallet *ConfluxWallet) GetBalance() (*big.Int, error) {
	return wallet.client.getBalance()
}

// GetUnconfirmedBalance returns the unconfirmed balance for the wallet
func (wallet *ConfluxWallet) GetUnconfirmedBalance() (*big.Int, error) {
	return wallet.client.getUnconfirmedBalance()
}

// Transfer will transfer the amount from this wallet to the spec address
func (wallet *ConfluxWallet) Transfer(to string, value *big.Int, spendAll bool, fee big.Int) (types.Hash, error) {
	networkID, _ := wallet.client.GetNetworkID()
	toAddress, err := cfxaddress.New(to, networkID)
	if err != nil {
		return "", err
	}

	val := hexutil.Big(*value)
	feeVal := hexutil.Big(fee)
	return wallet.client.Transfer(*wallet.address, toAddress, &val, spendAll, &feeVal)
}

// Start will start the wallet daemon
func (wallet *ConfluxWallet) Start() {
	done = make(chan bool)
	doneBalanceTicker = make(chan bool)
	// start the ticker to check for pending txn rcpts
	go func(wallet *ConfluxWallet) {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				// get the pending txns
				txns, err := wallet.db.Txns().GetAll(true)
				if err != nil {
					continue
				}
				for _, txn := range txns {
					hash := types.Hash(util.EnsureCorrectPrefix(txn.Txid))
					go func(txnData []byte) {
						_, err := wallet.checkTxnRcpt(hash, txnData)
						if err != nil {
							log.Errorf(err.Error())
						}
					}(txn.Bytes)
				}
			}
		}
	}(wallet)

	// start the ticker to check for balance
	go func(wallet *ConfluxWallet) {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		currentBalance, err := wallet.GetBalance()
		if err != nil {
			log.Infof("err fetching initial balance: %v", err)
		}
		currentTip, _ := wallet.ChainTip()
		if currentTip == 0 {
			log.Fatal("Failed to get chain tip when init")
		}

		for {
			select {
			case <-doneBalanceTicker:
				return
			case <-ticker.C:
				// fetch the current balance
				fetchedBalance, err := wallet.GetBalance()
				if err != nil {
					log.Infof("err fetching balance at %v: %v", time.Now(), err)
					continue
				}
				if fetchedBalance.Cmp(currentBalance) != 0 {
					// process balance change
					go wallet.processBalanceChange(currentBalance, fetchedBalance, currentTip)
					currentTip, _ = wallet.ChainTip()
					if currentTip == 0 {
						log.Warning("Failed to get chain tip")
						continue
					}
					currentBalance = fetchedBalance
				}
			}
		}
	}(wallet)
}

func (wallet *ConfluxWallet) processBalanceChange(previousBalance, currentBalance *big.Int, currentHead uint32) {
	count := 0
	cTip := int(currentHead)
	value := new(big.Int).Sub(currentBalance, previousBalance)
	for count < 30 {
		txns, err := wallet.TransactionsFromEpoch(&cTip)
		if err == nil && len(txns) > 0 {
			count = 30
			txncb := wi.TransactionCallback{
				Txid:      util.EnsureCorrectPrefix(txns[0].Txid),
				Outputs:   []wi.TransactionOutput{},
				Inputs:    []wi.TransactionInput{},
				Height:    txns[0].Height,
				Timestamp: time.Now(),
				Value:     *value,
				WatchOnly: false,
			}
			for _, l := range wallet.listeners {
				go l(txncb)
			}
			continue
		}

		time.Sleep(2 * time.Second)
		count++
	}
}

func (wallet *ConfluxWallet) invokeTxnCB(txnID string, value *big.Int) {
	txncb := wi.TransactionCallback{
		Txid:      util.EnsureCorrectPrefix(txnID),
		Outputs:   []wi.TransactionOutput{},
		Inputs:    []wi.TransactionInput{},
		Height:    0,
		Timestamp: time.Now(),
		Value:     *value,
		WatchOnly: false,
	}
	for _, l := range wallet.listeners {
		go l(txncb)
	}
}

// CurrencyCode returns CFX
func (wallet *ConfluxWallet) CurrencyCode() string {
	CurrencyCode := "CFX"
	if wallet.cfg.CoinType == wi.TestnetConflux {
		CurrencyCode = "TCFX"
	}

	return CurrencyCode
}

// IsDust Check if this amount is considered dust - 10000 drip
func (wallet *ConfluxWallet) IsDust(amount big.Int) bool {
	return amount.Cmp(big.NewInt(10000)) <= 0
}

// MasterPrivateKey - Get the master private key
func (wallet *ConfluxWallet) MasterPrivateKey() *hd.ExtendedKey {
	privateKey := wallet.getPrivateKey()

	defaultAccount, _ := wallet.am.GetDefault()
	commonAddr, _, _ := defaultAccount.ToCommon()

	return hd.NewExtendedKey([]byte{0x00, 0x00, 0x00, 0x00}, privateKey.D.Bytes(),
		commonAddr.Bytes(), commonAddr.Bytes(), 0, 0, true)
}

// MasterPublicKey - Get the master public key
func (wallet *ConfluxWallet) MasterPublicKey() *hd.ExtendedKey {
	privateKey := wallet.getPrivateKey()

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}

	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)

	defaultAccount, _ := wallet.am.GetDefault()
	commonAddr, _, _ := defaultAccount.ToCommon()
	return hd.NewExtendedKey([]byte{0x00, 0x00, 0x00, 0x00}, publicKeyBytes,
		commonAddr.Bytes(), commonAddr.Bytes(), 0, 0, false)
}

func (wallet *ConfluxWallet) ChildKey(keyBytes []byte, chaincode []byte, isPrivateKey bool) (*hd.ExtendedKey, error) {
	// TODO: Add REAL CHILD KEY of public or private key for conflux
	parentFP := []byte{0x00, 0x00, 0x00, 0x00}
	version := []byte{0x04, 0x88, 0xad, 0xe4} // starts with xprv
	if !isPrivateKey {
		version = []byte{0x04, 0x88, 0xb2, 0x1e}
	}
	// version := []byte{0x04, 0x35, 0x83, 0x94} // starts with xprv
	// if !isPrivateKey {
	// 	version = []byte{0x04, 0x35, 0x87, 0xcf}
	// }

	return hd.NewExtendedKey(version, keyBytes, chaincode, parentFP, 0, 0, isPrivateKey), nil
}

// CurrentAddress - Get the current address for the given purpose
func (wallet *ConfluxWallet) CurrentAddress(purpose wi.KeyPurpose) btcutil.Address {
	commonAddr := wallet.address.MustGetCommonAddress()
	return EthAddress{&commonAddr}
}

// NewAddress - Returns a fresh address that has never been returned by this function
func (wallet *ConfluxWallet) NewAddress(purpose wi.KeyPurpose) btcutil.Address {
	return wallet.CurrentAddress(purpose)
}

// DecodeAddress - Parse the address string and return an address interface
func (wallet *ConfluxWallet) DecodeAddress(addr string) (btcutil.Address, error) {
	if len(addr) > 64 {
		commonAddr, err := cfxScriptToAddr(addr)
		if err != nil {
			log.Error(err.Error())
		}
		return EthAddress{&commonAddr}, err
	}

	cfxAddr, err := cfxaddress.NewFromBase32(addr)
	if err == nil {
		commonAddr, _, _ := cfxAddr.ToCommon()
		return EthAddress{&commonAddr}, err
	}

	commonAddr := common.HexToAddress(addr)
	return EthAddress{&commonAddr}, nil
}

func cfxScriptToAddr(addr string) (common.Address, error) {
	rScriptBytes, err := hex.DecodeString(addr)
	if err != nil {
		return common.Address{}, err
	}
	rScript, err := DeserializeCfxScript(rScriptBytes)
	if err != nil {
		return common.Address{}, err
	}
	_, sHash, err := GenScriptHash(rScript)
	if err != nil {
		return common.Address{}, err
	}
	return common.HexToAddress(sHash), nil
}

// ScriptToAddress - ?
func (wallet *ConfluxWallet) ScriptToAddress(script []byte) (btcutil.Address, error) {
	return wallet.CurrentAddress(0), nil
}

// HasKey - Returns if the wallet has the key for the given address
func (wallet *ConfluxWallet) HasKey(addr btcutil.Address) bool {
	if !util.IsValidAddress(addr.String()) {
		return false
	}
	return wallet.address.String() == addr.String()
}

// Balance - Get the confirmed and unconfirmed balances
func (wallet *ConfluxWallet) Balance() (confirmed, unconfirmed wi.CurrencyValue) {
	var balance, ucbalance wi.CurrencyValue
	bal, err := wallet.GetBalance()
	if err == nil {
		if bal == nil {
			bal = big.NewInt(0)
		}

		balance = wi.CurrencyValue{
			Value:    *bal,
			Currency: CfxCurrencyDefinition,
		}
	}

	if bal == nil {
		bal = big.NewInt(0)
	}
	ucbal, err := wallet.GetUnconfirmedBalance()
	ucb := big.NewInt(0)
	if err == nil && ucbal != nil {
		if ucbal.Cmp(bal) > 0 {
			ucb.Sub(ucbal, bal)
		}
	}
	ucbalance = wi.CurrencyValue{
		Value:    *ucb,
		Currency: CfxCurrencyDefinition,
	}
	return balance, ucbalance
}

func (wallet *ConfluxWallet) equalsCfxAddress(addr1 string, addr2 string) bool {
	if addr1 == addr2 {
		return true
	}

	networkID, _ := wallet.client.GetNetworkID()
	cfxAddress1, err := cfxaddress.New(addr1, networkID)
	if err != nil {
		return false
	}
	cfxAddress2, err := cfxaddress.New(addr2, networkID)
	if err != nil {
		return false
	}

	return cfxAddress1.Equals(&cfxAddress2)
}

// TransactionsFromEpoch - Returns a list of transactions for this wallet begining from the specified epoch
func (wallet *ConfluxWallet) TransactionsFromEpoch(startBlock *int) ([]wi.Txn, error) {
	ret := []wi.Txn{}

	unconf, _ := wallet.db.Txns().GetAll(false)

	txns, err := wallet.client.scaner.getTxRecords(wallet.address.String(), 0, 50)
	if err != nil && len(unconf) == 0 {
		log.Error("err fetching transactions : ", err)
		return []wi.Txn{}, nil
	}

	currentTip, _ := wallet.ChainTip()
	if currentTip == 0 {
		log.Error("err failed to get ChainTip")
		return []wi.Txn{}, nil
	}
	for i := len(txns) - 1; i >= 0; i-- {
		t := txns[i]
		status := wi.StatusConfirmed
		prefix := ""
		if t.Status == 1 {
			status = wi.StatusError
		}

		if wallet.equalsCfxAddress(t.From, wallet.address.String()) {
			prefix = "-"
		}

		val := t.Value

		val = prefix + val

		tnew := wi.Txn{
			Txid:          util.EnsureCorrectPrefix(t.Hash),
			Value:         val,
			Height:        int32(t.EpochNumber),
			Timestamp:     time.Unix(t.Timestamp, 0),
			WatchOnly:     false,
			Confirmations: int64(currentTip - t.EpochNumber),
			Status:        wi.StatusCode(status),
			Bytes:         []byte{},
		}
		ret = append(ret, tnew)
	}

	for _, u := range unconf {
		u.Status = wi.StatusUnconfirmed
		ret = append(ret, u)
	}

	return ret, nil
}

// Transactions - Returns a list of transactions for this wallet
func (wallet *ConfluxWallet) Transactions() ([]wi.Txn, error) {
	return wallet.TransactionsFromEpoch(nil)
}

// GetTransaction - Get info on a specific transaction
func (wallet *ConfluxWallet) GetTransaction(txid chainhash.Hash) (wi.Txn, error) {
	tx, err := wallet.client.GetTransactionByHash(types.Hash(util.EnsureCorrectPrefix(txid.String())))
	// From testing, when buy goods and the vendor node is in another country, it usually happens err is nil but tx is nil.
	// when manually query in https://testnet.confluxscan.io/, the tx exists.
	for i := 0; i < 10; i++ {
		if tx != nil {
			break
		}

		time.Sleep(1 * time.Second)
		tx, err = wallet.client.GetTransactionByHash(types.Hash(util.EnsureCorrectPrefix(txid.String())))
	}

	if err != nil {
		return wi.Txn{}, err
	}

	if tx == nil {
		return wi.Txn{}, errors.New("tx is nil!!!, txid: " + txid.String())
	}

	fromAddr := tx.From
	toAddr := tx.To
	valueSub := big.NewInt(5000000)

	v, err := wallet.registry.GetRecommendedVersion(nil, "escrow")
	if err == nil {
		if tx.To.String() == v.Implementation.String() {
			toAddr = wallet.address
		}
		if tx.Value.ToInt().Cmp(valueSub) > 0 {
			valueSub = tx.Value.ToInt()
		}
	}

	commonToAddr := toAddr.MustGetCommonAddress()
	commonFromAddr := fromAddr.MustGetCommonAddress()
	return wi.Txn{
		Txid:        tx.Hash.String(),
		Value:       tx.Value.ToInt().String(),
		Height:      0,
		Timestamp:   time.Now(),
		WatchOnly:   false,
		Bytes:       []byte(tx.Data),
		ToAddress:   commonToAddr.String(),
		FromAddress: commonFromAddr.String(),
		Outputs: []wi.TransactionOutput{
			{
				Address: EthAddress{&commonToAddr},
				Value:   *valueSub,
				Index:   0,
			},
			{
				Address: EthAddress{&commonFromAddr},
				Value:   *valueSub,
				Index:   1,
			},
			{
				Address: EthAddress{&commonToAddr},
				Value:   *valueSub,
				Index:   2,
			},
		},
	}, nil
}

// ChainTip - Get the height and best hash of the blockchain
func (wallet *ConfluxWallet) ChainTip() (uint32, chainhash.Hash) {
	status, err := wallet.client.GetStatus()
	if err != nil {
		fmt.Printf("- get status error: %v\n\n", err.Error())
		return 0, *emptyChainHash
	}

	h, err := util.CreateChainHash(status.BestHash.String())
	if err != nil {
		log.Error(err.Error())
		return 0, *emptyChainHash
	}
	return uint32(status.EpochNumber), *h
}

// GetFeePerByte - Get the current fee per byte
func (wallet *ConfluxWallet) GetFeePerByte(feeLevel wi.FeeLevel) big.Int {
	return wallet.client.GetFeePerByte(feeLevel)
}

func (wallet *ConfluxWallet) getPrivateKey() *ecdsa.PrivateKey {
	defaultAccount, _ := wallet.am.GetDefault()
	keyString, _ := wallet.am.Export(*defaultAccount, "")
	if utils.Has0xPrefix(keyString) {
		keyString = keyString[2:]
	}
	privateKey, _ := crypto.HexToECDSA(keyString)

	return privateKey
}

func (wallet *ConfluxWallet) getSpendGas(amount big.Int, addr btcutil.Address, feeLevel wi.FeeLevel) big.Int {
	networkID, _ := wallet.client.GetNetworkID()
	from := wallet.address
	to, err := cfxaddress.New(addr.String(), networkID)
	if err != nil {
		big.NewInt(0)
	}

	val := hexutil.Big(amount)
	gas, err := wallet.client.EstimateTxnGas(*from, to, &val)
	if err != nil || gas == nil {
		big.NewInt(0)
	}

	return *gas.ToInt()
}

// Spend - Send conflux to an external wallet
func (wallet *ConfluxWallet) Spend(amount big.Int, addr btcutil.Address, feeLevel wi.FeeLevel, referenceID string, spendAll bool) (*chainhash.Hash, error) {
	var (
		hash      types.Hash
		h         *chainhash.Hash
		watchOnly bool
		nonce     int32
		err       error
	)
	actualRecipient := addr

	if referenceID == "" {
		if !wallet.client.balanceCheck(feeLevel, amount) {
			return nil, wi.ErrInsufficientFunds
		}

		// no referenceID means this is a direct transfer
		hash, err = wallet.Transfer(addr.String(), &amount, spendAll, wallet.GetFeePerByte(feeLevel))
	} else {
		watchOnly = true
		// this is a spend which means it has to be linked to an order
		// specified using the referenceID

		// check if the addr is a multisig addr
		scripts, err := wallet.db.WatchedScripts().GetAll()
		if err != nil {
			return nil, err
		}
		isScript := false
		addrCfx := common.HexToAddress(addr.String())
		key := addrCfx.Bytes()
		redeemScript := []byte{}

		for _, script := range scripts {
			if bytes.Equal(key, script[:common.AddressLength]) {
				isScript = true
				redeemScript = script[common.AddressLength:]
				break
			}
		}

		if isScript {
			cfxScript, err := DeserializeCfxScript(redeemScript)
			if err != nil {
				return nil, err
			}
			_, scrHash, err := GenScriptHash(cfxScript)
			if err != nil {
				log.Error(err.Error())
			}
			addrScrHash := common.HexToAddress(scrHash)
			actualRecipient = EthAddress{address: &addrScrHash}
			hash, _, err = wallet.callAddTransaction(cfxScript, &amount, feeLevel)
			if err != nil {
				log.Errorf("error call add txn: %v", err)
				return nil, wi.ErrInsufficientFunds
			}
		} else {
			if !wallet.client.balanceCheck(feeLevel, amount) {
				return nil, wi.ErrInsufficientFunds
			}
			hash, err = wallet.Transfer(addr.String(), &amount, spendAll, wallet.GetFeePerByte(feeLevel))
		}
		if err != nil {
			return nil, err
		}

		// txn is pending
		nonce, err = wallet.client.GetTxnNonce(util.EnsureCorrectPrefix(hash.String()))
		if err != nil {
			return nil, err
		}
	}
	if err == nil {
		h, err = util.CreateChainHash(hash.String())
		if err == nil {
			wallet.invokeTxnCB(h.String(), &amount)
		}
	}
	commonFrom, _ := wallet.address.ToHex()
	data, err := SerializePendingTxn(PendingTxn{
		TxnID:     hash,
		Amount:    amount.String(),
		OrderID:   referenceID,
		Nonce:     nonce,
		From:      commonFrom,
		To:        actualRecipient.EncodeAddress(),
		WithInput: false,
	})
	if err == nil {
		err0 := wallet.db.Txns().Put(data, ut.NormalizeAddress(hash.String()), amount.String(), 0, time.Now(), watchOnly)
		if err0 != nil {
			log.Error(err0.Error())
		}
	}
	return h, nil
}

func (wallet *ConfluxWallet) createTxnCallback(txID, orderID string, toAddress btcutil.Address, value big.Int, bTime time.Time, withInput bool, height int64) wi.TransactionCallback {
	output := wi.TransactionOutput{
		Address: toAddress,
		Value:   value,
		Index:   1,
		OrderID: orderID,
	}

	input := wi.TransactionInput{}

	if withInput {
		input = wi.TransactionInput{
			OutpointHash:  []byte(util.EnsureCorrectPrefix(txID)),
			OutpointIndex: 1,
			LinkedAddress: toAddress,
			Value:         value,
			OrderID:       orderID,
		}

	}
	return wi.TransactionCallback{
		Txid:      util.EnsureCorrectPrefix(txID),
		Outputs:   []wi.TransactionOutput{output},
		Inputs:    []wi.TransactionInput{input},
		Height:    int32(height),
		Timestamp: time.Now(),
		Value:     value,
		WatchOnly: false,
		BlockTime: bTime,
	}
}

func (wallet *ConfluxWallet) AssociateTransactionWithOrder(txnCB wi.TransactionCallback) {
	for _, l := range wallet.listeners {
		go l(txnCB)
	}
}

// checkTxnRcpt check the txn rcpt status
func (wallet *ConfluxWallet) checkTxnRcpt(hash types.Hash, data []byte) (*types.Hash, error) {
	pTxn, err := DeserializePendingTxn(data)
	if err != nil {
		return nil, err
	}

	tx, err := wallet.client.GetTransactionByHash(hash)
	if err != nil {
		log.Infof("fetching txn rcpt: %v", err)
	}

	if tx != nil {
		// good. so the txn has been processed but we have to account for failed
		// but valid txn like some contract condition causing revert
		if tx.Status != nil {
			// all good to update order state
			chash, err := util.CreateChainHash(hash.String())
			if err != nil {
				return nil, err
			}
			err = wallet.db.Txns().Delete(chash)
			if err != nil {
				log.Errorf("err deleting the pending txn : %v", err)
			}
			n := new(big.Int)
			n, _ = n.SetString(pTxn.Amount, 10)
			withInput := true
			if pTxn.Amount != "0" {
				withInput = pTxn.WithInput
			}
			toAddr, _ := wallet.DecodeAddress(pTxn.To)
			height := tx.EpochHeight.ToInt().Int64()
			go wallet.AssociateTransactionWithOrder(
				wallet.createTxnCallback(hash.String(), pTxn.OrderID, toAddr,
					*n, time.Now(), withInput, height))
		}
	}
	return &hash, nil

}

// BumpFee - Bump the fee for the given transaction
func (wallet *ConfluxWallet) BumpFee(txid chainhash.Hash) (*chainhash.Hash, error) {
	return util.CreateChainHash(txid.String())
}

// EstimateFee - Calculates the estimated size of the transaction and returns the total fee for the given feePerByte
func (wallet *ConfluxWallet) EstimateFee(ins []wi.TransactionInput, outs []wi.TransactionOutput, feePerByte big.Int) big.Int {
	sum := big.NewInt(0)
	for _, out := range outs {
		from := wallet.address
		to, err := wallet.DecodeAddress(out.Address.String())
		if err != nil {
			return *sum
		}

		toCommon := to.(*EthAddress).address
		networkID, _ := wallet.client.GetNetworkID()
		toCfx, err := cfxaddress.NewFromCommon(*toCommon, networkID)
		if err != nil {
			return *sum
		}

		val := hexutil.Big(out.Value)
		gas, err := wallet.client.EstimateTxnGas(*from, toCfx, &val)
		if err != nil {
			return *sum
		}
		sum.Add(sum, gas.ToInt())
	}
	return *sum
}

// EstimateSpendFee - Build a spend transaction for the amount and return the transaction fee
func (wallet *ConfluxWallet) EstimateSpendFee(amount big.Int, feeLevel wi.FeeLevel) (big.Int, error) {
	if !wallet.client.balanceCheck(feeLevel, amount) {
		return *big.NewInt(0), wi.ErrInsufficientFunds
	}

	val := hexutil.Big(amount)
	return wallet.client.EstimateGasSpend(*wallet.address, &val)
}

// SweepAddress - Build and broadcast a transaction that sweeps all coins from an address. If it is a p2sh multisig, the redeemScript must be included
func (wallet *ConfluxWallet) SweepAddress(utxos []wi.TransactionInput, address *btcutil.Address, key *hd.ExtendedKey, redeemScript *[]byte, feeLevel wi.FeeLevel) (*chainhash.Hash, error) {

	addr := wallet.CurrentAddress(0)

	outs := []wi.TransactionOutput{}
	for i, in := range utxos {
		out := wi.TransactionOutput{
			Address: addr,
			Value:   in.Value,
			Index:   uint32(i),
			OrderID: in.OrderID,
		}
		outs = append(outs, out)
	}

	sigs, err := wallet.CreateMultisigSignature([]wi.TransactionInput{}, outs, key, *redeemScript, *big.NewInt(1))
	if err != nil {
		return nil, err
	}

	data, err := wallet.Multisign([]wi.TransactionInput{}, outs, sigs, []wi.Signature{}, *redeemScript, *big.NewInt(1), false)
	if err != nil {
		return nil, err
	}
	hash := common.BytesToHash(data)

	return util.CreateChainHash(hash.Hex())
}

// ExchangeRates - return the exchangerates
func (wallet *ConfluxWallet) ExchangeRates() wi.ExchangeRates {
	return wallet.exchangeRates
}

func (wallet *ConfluxWallet) callAddTransaction(script CfxRedeemScript, value *big.Int, feeLevel wi.FeeLevel) (types.Hash, uint64, error) {
	// call registry to get the deployed address for the escrow ct
	fromAddress := wallet.address
	nonce, err := wallet.client.GetNextNonce(*fromAddress, types.EpochLatestState)
	if err != nil {
		log.Fatal(err)
		log.Errorf("err get next nonce: %v", err)
		return "", 0, err
	}
	gasPrice, err := wallet.client.GetGasPrice()
	if err != nil {
		log.Fatal(err)
		log.Errorf("err get gas price: %v", err)
		return "", 0, err
	}
	gasPriceCFXGAS := wallet.GetFeePerByte(feeLevel)
	if gasPriceCFXGAS.Int64() < gasPrice.ToInt().Int64() {
		gasPriceCFXGAS = *gasPrice.ToInt()
	}

	val := hexutil.Big(*value)
	transactOpts := &bind.TransactOpts{
		Nonce:    nonce,
		Value:    &val, // in wei
		GasPrice: gasPrice,
	}

	// lets check if the caller has enough balance to make the
	// multisign call
	if !wallet.client.balanceCheck(feeLevel, *value) {
		// the wallet does not have the required balance
		return "", nonce.ToInt().Uint64(), wi.ErrInsufficientFunds
	}

	shash, _, err := GenScriptHash(script)
	if err != nil {
		return "", nonce.ToInt().Uint64(), err
	}

	networkID, _ := wallet.client.GetNetworkID()
	multisigAddress, err := cfxaddress.NewFromCommon(script.MultisigAddress, networkID)
	if err != nil {
		log.Errorf("error converting multi sign address: %s", err.Error())
		return "", nonce.ToInt().Uint64(), err
	}

	smtct, err := NewEscrow(multisigAddress, wallet.client)
	if err != nil {
		log.Errorf("error initilaizing contract failed: %s", err.Error())
		return "", nonce.ToInt().Uint64(), err
	}

	err = wallet.client.AccountManager.UnlockDefault("")
	if err != nil {
		return "", nonce.ToInt().Uint64(), errors.New("account manager failed to unlock default address")
	}

	var tx *types.UnsignedTransaction
	var h *types.Hash
	tx, h, err = smtct.AddTransaction(transactOpts, script.Buyer, script.Seller,
		script.Moderator, script.Threshold, script.Timeout, shash, script.TxnID)
	if err != nil {
		return "", 0, err
	}

	txns = append(txns, wi.Txn{
		Txid:      h.String(),
		Value:     value.String(),
		Height:    int32(nonce.ToInt().Int64()),
		Timestamp: time.Now(),
		WatchOnly: false,
		Bytes:     tx.Data})

	return *h, nonce.ToInt().Uint64(), err

}

// GenerateMultisigScript - Generate a multisig script from public keys. If a timeout is included the returned script should be a timelocked escrow which releases using the timeoutKey.
func (wallet *ConfluxWallet) GenerateMultisigScript(keys []hd.ExtendedKey, threshold int, timeout time.Duration, timeoutKey *hd.ExtendedKey) (btcutil.Address, []byte, error) {
	if uint32(timeout.Hours()) > 0 && timeoutKey == nil {
		return nil, nil, errors.New("timeout key must be non nil when using an escrow timeout")
	}

	if len(keys) < threshold {
		return nil, nil, fmt.Errorf("unable to generate multisig script with "+
			"%d required signatures when there are only %d public "+
			"keys available", threshold, len(keys))
	}

	if len(keys) < 2 {
		return nil, nil, fmt.Errorf("unable to generate multisig script with "+
			"%d required signatures when there are only %d public "+
			"keys available", threshold, len(keys))
	}

	var ecKeys []common.Address
	for _, key := range keys {
		ecKey, err := key.ECPubKey()
		if err != nil {
			return nil, nil, err
		}
		ePubkey := ecKey.ToECDSA()

		addr := util.EnsureCorrectPrefix(crypto.PubkeyToAddress(*ePubkey).Hex())
		addr = "0x1" + addr[3:]
		commonAddr := common.HexToAddress(addr)

		ecKeys = append(ecKeys, commonAddr)
	}

	ver, err := wallet.registry.GetRecommendedVersion(nil, "escrow")
	if err != nil {
		log.Fatal(err)
	}

	if util.IsZeroAddress(ver.Implementation) {
		return nil, nil, errors.New("no escrow contract available")
	}

	builder := CfxRedeemScript{}

	builder.TxnID = common.BytesToAddress(util.ExtractChaincode(&keys[0]))
	builder.Timeout = uint32(timeout.Hours())
	builder.Threshold = uint8(threshold)
	builder.Buyer = ecKeys[0]
	builder.Seller = ecKeys[1]
	builder.MultisigAddress = ver.Implementation

	if threshold > 1 {
		builder.Moderator = ecKeys[2]
	}
	switch threshold {
	case 1:
		{
			// Seller is offline
		}
	case 2:
		{
			// Moderated payment
		}
	default:
		{
			// handle this
		}
	}

	redeemScript, err := SerializeCfxScript(builder)
	if err != nil {
		return nil, nil, err
	}

	addr := common.HexToAddress(hexutil.Encode(crypto.Keccak256(redeemScript))) //hash.Sum(nil)[:]))
	retAddr := EthAddress{&addr}

	scriptKey := append(addr.Bytes(), redeemScript...)
	err = wallet.db.WatchedScripts().Put(scriptKey)
	if err != nil {
		log.Errorf("err saving the redeemscript: %v", err)
	}

	return retAddr, redeemScript, nil
}

// CreateMultisigSignature - Create a signature for a multisig transaction
func (wallet *ConfluxWallet) CreateMultisigSignature(ins []wi.TransactionInput, outs []wi.TransactionOutput, key *hd.ExtendedKey, redeemScript []byte, feePerByte big.Int) ([]wi.Signature, error) {

	payouts := []wi.TransactionOutput{}
	difference := new(big.Int)

	if len(ins) > 0 {
		totalVal := ins[0].Value
		outVal := new(big.Int)
		for _, out := range outs {
			outVal = new(big.Int).Add(outVal, &out.Value)
		}
		if totalVal.Cmp(outVal) != 0 {
			if totalVal.Cmp(outVal) < 0 {
				return nil, errors.New("payout greater than initial amount")
			}
			difference = new(big.Int).Sub(&totalVal, outVal)
		}
	}

	rScript, err := DeserializeCfxScript(redeemScript)
	if err != nil {
		return nil, err
	}

	indx := []int{}
	mbvAddresses := make([]string, 3)

	for i, out := range outs {
		if out.Value.Cmp(new(big.Int)) > 0 {
			indx = append(indx, i)
		}
		if out.Address.String() == rScript.Moderator.Hex() {
			mbvAddresses[0] = out.Address.String()
		} else if out.Address.String() == rScript.Buyer.Hex() && (out.Value.Cmp(new(big.Int)) > 0) {
			mbvAddresses[1] = out.Address.String()
		} else {
			mbvAddresses[2] = out.Address.String()
		}
		p := wi.TransactionOutput{
			Address: out.Address,
			Value:   out.Value,
			Index:   out.Index,
			OrderID: out.OrderID,
		}
		payouts = append(payouts, p)
	}

	if len(indx) > 0 {
		diff := new(big.Int)
		delta := new(big.Int)
		diff.DivMod(difference, big.NewInt(int64(len(indx))), delta)
		for _, i := range indx {
			payouts[i].Value.Add(&payouts[i].Value, diff)
		}
		payouts[indx[0]].Value.Add(&payouts[indx[0]].Value, delta)
	}

	sort.Slice(payouts, func(i, j int) bool {
		return strings.Compare(payouts[i].Address.String(), payouts[j].Address.String()) == -1
	})

	var sigs []wi.Signature

	payables := make(map[string]big.Int)
	addresses := []string{}
	for _, out := range payouts {
		if out.Value.Cmp(big.NewInt(0)) <= 0 {
			continue
		}
		val := new(big.Int).SetBytes(out.Value.Bytes()) // &out.Value
		if p, ok := payables[out.Address.String()]; ok {
			sum := new(big.Int).Add(val, &p)
			payables[out.Address.String()] = *sum
		} else {
			payables[out.Address.String()] = *val
			addresses = append(addresses, out.Address.String())
		}
	}

	sort.Strings(addresses)
	destArr := []byte{}
	amountArr := []byte{}

	for _, k := range mbvAddresses {
		v := payables[k]
		fmt.Printf("\nThe address is: %v, amount is: %v", k, v.Int64())
		if v.Cmp(big.NewInt(0)) != 1 {
			continue
		}
		addr := common.HexToAddress(k)
		sample := [32]byte{}
		sampleDest := [32]byte{}
		copy(sampleDest[12:], addr.Bytes())
		val := v.Bytes()
		l := len(val)

		copy(sample[32-l:], val)
		destArr = append(destArr, sampleDest[:]...)
		amountArr = append(amountArr, sample[:]...)
	}

	shash, _, err := GenScriptHash(rScript)
	if err != nil {
		return nil, err
	}

	var txHash [32]byte
	var payloadHash [32]byte

	/*
				// Follows ERC191 signature scheme: https://github.com/ethereum/EIPs/issues/191
		        bytes32 txHash = keccak256(
		            abi.encodePacked(
		                "\x19Ethereum Signed Message:\n32",
		                keccak256(
		                    abi.encodePacked(
		                        byte(0x19),
		                        byte(0),
		                        this,
		                        destinations,
		                        amounts,
		                        scriptHash
		                    )
		                )
		            )
		        );

	*/

	payload := []byte{byte(0x19), byte(0)}
	payload = append(payload, rScript.MultisigAddress.Bytes()...)
	// payload = append(payload, destArr...)
	// payload = append(payload, amountArr...)
	payload = append(payload, shash[:]...)

	pHash := crypto.Keccak256(payload)
	copy(payloadHash[:], pHash)

	txData := []byte{byte(0x19)}
	txData = append(txData, []byte("Ethereum Signed Message:\n32")...)
	txData = append(txData, payloadHash[:]...)
	txnHash := crypto.Keccak256(txData)
	log.Debugf("txnHash        : %s", hexutil.Encode(txnHash))
	log.Debugf("phash          : %s", hexutil.Encode(payloadHash[:]))
	copy(txHash[:], txnHash)

	privateKey := wallet.getPrivateKey()
	sig, err := crypto.Sign(txHash[:], privateKey)
	if err != nil {
		log.Errorf("error signing in createmultisig : %v", err)
	}
	sigs = append(sigs, wi.Signature{InputIndex: 1, Signature: sig})

	return sigs, err
}

// Multisign - Combine signatures and optionally broadcast
func (wallet *ConfluxWallet) Multisign(ins []wi.TransactionInput, outs []wi.TransactionOutput, sigs1 []wi.Signature, sigs2 []wi.Signature, redeemScript []byte, feePerByte big.Int, broadcast bool) ([]byte, error) {

	payouts := []wi.TransactionOutput{}
	difference := new(big.Int)

	if len(ins) > 0 {
		totalVal := &ins[0].Value
		outVal := new(big.Int)
		for _, out := range outs {
			outVal.Add(outVal, &out.Value)
		}
		if totalVal.Cmp(outVal) != 0 {
			if totalVal.Cmp(outVal) < 0 {
				return nil, errors.New("payout greater than initial amount")
			}
			difference.Sub(totalVal, outVal)
		}
	}

	rScript, err := DeserializeCfxScript(redeemScript)
	if err != nil {
		return nil, err
	}

	indx := []int{}
	referenceID := ""
	mbvAddresses := make([]string, 3)

	for i, out := range outs {
		if out.Value.Cmp(new(big.Int)) > 0 {
			indx = append(indx, i)
		}
		if out.Address.String() == rScript.Moderator.Hex() {
			indx = append(indx, i)
			mbvAddresses[0] = out.Address.String()
		} else if out.Address.String() == rScript.Buyer.Hex() {
			mbvAddresses[1] = out.Address.String()
		} else {
			mbvAddresses[2] = out.Address.String()
		}
		p := wi.TransactionOutput{
			Address: out.Address,
			Value:   out.Value,
			Index:   out.Index,
			OrderID: out.OrderID,
		}
		referenceID = out.OrderID
		payouts = append(payouts, p)
	}

	if len(indx) > 0 {
		diff := new(big.Int)
		delta := new(big.Int)
		diff.DivMod(difference, big.NewInt(int64(len(indx))), delta)
		for _, i := range indx {
			payouts[i].Value.Add(&payouts[i].Value, diff)
		}
		payouts[indx[0]].Value.Add(&payouts[indx[0]].Value, delta)
	}

	sort.Slice(payouts, func(i, j int) bool {
		return strings.Compare(payouts[i].Address.String(), payouts[j].Address.String()) == -1
	})

	payables := make(map[string]big.Int)
	for _, out := range payouts {
		if out.Value.Cmp(big.NewInt(0)) <= 0 {
			continue
		}
		val := new(big.Int).SetBytes(out.Value.Bytes()) // &out.Value
		if p, ok := payables[out.Address.String()]; ok {
			sum := new(big.Int).Add(val, &p)
			payables[out.Address.String()] = *sum
		} else {
			payables[out.Address.String()] = *val
		}
	}

	rSlice := [][32]byte{}
	sSlice := [][32]byte{}
	vSlice := []uint8{}

	var r [32]byte
	var s [32]byte
	var v uint8

	if len(sigs1) > 0 && len(sigs1[0].Signature) > 0 {
		r, s, v = util.SigRSV(sigs1[0].Signature)
		rSlice = append(rSlice, r)
		sSlice = append(sSlice, s)
		vSlice = append(vSlice, v)
	}

	if len(sigs2) > 0 && len(sigs2[0].Signature) > 0 {
		r, s, v = util.SigRSV(sigs2[0].Signature)
		rSlice = append(rSlice, r)
		sSlice = append(sSlice, s)
		vSlice = append(vSlice, v)
	}

	shash, _, err := GenScriptHash(rScript)
	if err != nil {
		return nil, err
	}

	networkID, _ := wallet.client.GetNetworkID()
	multisigAddress, err := cfxaddress.NewFromCommon(rScript.MultisigAddress, networkID)
	if err != nil {
		log.Fatalf("error converting multi sign address: %s", err.Error())
	}
	smtct, err := NewEscrow(multisigAddress, wallet.client)
	if err != nil {
		log.Fatalf("error initializing contract failed: %s", err.Error())
	}

	destinations := []common.Address{}
	amounts := []*big.Int{}

	for _, k := range mbvAddresses {
		v := payables[k]
		if v.Cmp(big.NewInt(0)) == 1 {
			destinations = append(destinations, common.HexToAddress(k))
			amounts = append(amounts, new(big.Int).SetBytes(v.Bytes()))
		}
	}

	fromAddress := wallet.address
	nonce, err := wallet.client.GetNextNonce(*fromAddress, types.EpochLatestState)
	if err != nil {
		log.Fatal(err)
	}
	gasPrice, err := wallet.client.GetGasPrice()
	if err != nil {
		log.Fatal(err)
	}

	transactOpts := &bind.TransactOpts{
		Nonce:    nonce,
		GasPrice: gasPrice,
	}

	// lets check if the caller has enough balance to make the
	// multisign call
	requiredBalance := new(big.Int).Mul(gasPrice.ToInt(), big.NewInt(maxGasLimit))
	currentBalance, err := wallet.GetBalance()
	if err != nil {
		log.Error("err fetching cfx wallet balance")
		currentBalance = big.NewInt(0)
	}

	if requiredBalance.Cmp(currentBalance) > 0 {
		// the wallet does not have the required balance
		return nil, wi.ErrInsufficientFunds
	}

	err = wallet.client.AccountManager.UnlockDefault("")
	if err != nil {
		return nil, errors.New("account manager failed to unlock default address")
	}

	tx, hash, txnErr := smtct.Execute(transactOpts, vSlice, rSlice, sSlice, shash, destinations, amounts)

	if txnErr != nil {
		return nil, txnErr
	}

	txns = append(txns, wi.Txn{
		Txid:      hash.String(),
		Value:     "0",
		Height:    int32(nonce.ToInt().Int64()),
		Timestamp: time.Now(),
		WatchOnly: false,
		Bytes:     tx.Data})

	// this is a pending txn
	_, scrHash, err := GenScriptHash(rScript)
	if err != nil {
		log.Error(err.Error())
	}

	commonFrom, _ := wallet.address.ToHex()
	data, err := SerializePendingTxn(PendingTxn{
		TxnID:   *hash,
		Amount:  "0",
		OrderID: referenceID,
		Nonce:   int32(nonce.ToInt().Int64()),
		From:    commonFrom,
		To:      scrHash,
	})
	if err == nil {
		err0 := wallet.db.Txns().Put(data, ut.NormalizeAddress(hash.String()), "0", 0, time.Now(), true)
		if err0 != nil {
			log.Error(err0.Error())
		}
	}

	return hash.ToCommonHash().Bytes(), nil
}

// AddWatchedAddresses - Add a script to the wallet and get notifications back when coins are received or spent from it
func (wallet *ConfluxWallet) AddWatchedAddresses(addrs ...btcutil.Address) error {
	// the reason cfx wallet cannot use this as of now is because only the address
	// is insufficient, the redeemScript is also required
	return nil
}

// AddTransactionListener will call the function callback when new transactions are discovered
func (wallet *ConfluxWallet) AddTransactionListener(callback func(wi.TransactionCallback)) {
	// add incoming txn listener using service
	wallet.listeners = append(wallet.listeners, callback)
}

// ReSyncBlockchain - Use this to re-download merkle blocks in case of missed transactions
func (wallet *ConfluxWallet) ReSyncBlockchain(fromTime time.Time) {
	// use service here
}

// GetConfirmations - Return the number of confirmations and the height for a transaction
func (wallet *ConfluxWallet) GetConfirmations(txid chainhash.Hash) (confirms, atHeight uint32, err error) {
	tx, err := wallet.client.GetTransactionByHash(types.Hash(util.EnsureCorrectPrefix(txid.String())))
	if err != nil {
		return 0, 0, err
	}

	highestEpoch, err := wallet.client.GetEpochNumber()

	ucfs := big.NewInt(0)
	ucfs.Sub(highestEpoch.ToInt(), tx.EpochHeight.ToInt())

	return uint32(ucfs.Uint64()), uint32(tx.EpochHeight.ToInt().Uint64()), nil
}

// Close will stop the wallet daemon
func (wallet *ConfluxWallet) Close() {
	// stop the wallet daemon
	done <- true
	doneBalanceTicker <- true
}

// GenDefaultKeyStore will generate a default keystore
// func GenDefaultKeyStore(passwd string) (*Account, error) {
// 	ks := keystore.NewKeyStore("./", keystore.StandardScryptN, keystore.StandardScryptP)
// 	account, err := ks.NewAccount(passwd)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return NewAccountFromKeyfile(account.URL.Path, passwd)
// }
