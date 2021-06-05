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

	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	"github.com/Conflux-Chain/go-conflux-sdk/utils"
	"github.com/OpenBazaar/go-ethwallet/util"
	mwConfig "github.com/OpenBazaar/multiwallet/config"
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
	TxnID     common.Hash
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
	account *Account
	am      *sdk.AccountManager
	address *CfxAddress

	db            wi.Datastore
	exchangeRates wi.ExchangeRates
	listeners     []func(wi.TransactionCallback)
}

// NewConfluxWallet will return a reference to the Cfx Wallet
func NewConfluxWallet(cfg mwConfig.CoinConfig, mnemonic string, repoPath string, proxy proxy.Dialer) (*ConfluxWallet, error) {
	url := "https://test.confluxrpc.com"

	// var networkID uint32 = 1029
	// if cfg.CoinType == wi.TestnetConflux {
	// 	networkID = 1
	// }

	// privateKey, err := getPrivateKey(mnemonic)
	// if err != nil {
	// 	log.Errorf("get private key from mnemonic failed: %s", err.Error())
	// 	return nil, err
	// }

	keyStorePath := path.Join(repoPath, "keystore")
	// am := sdk.NewAccountManager(keyStorePath, networkID)
	// address, err := am.ImportKey(privateKey, "")
	// if err != nil {
	// 	log.Errorf("import key failed: %s", err.Error())
	// 	return nil, err
	// }
	// log.Infof("The address is: %v", address.String())

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

	account, _ := NewAccountFromMnemonic(mnemonic, "")

	er := NewConfluxPriceFetcher(proxy)

	return &ConfluxWallet{cfg, client, account, nil, &CfxAddress{address}, cfg.DB, er, []func(wi.TransactionCallback){}}, nil
}

// Close will stop the wallet daemon
func (wallet *ConfluxWallet) Close() {
	// stop the wallet daemon
	done <- true
	doneBalanceTicker <- true
}

// Params - return nil to comply
func (wallet *ConfluxWallet) Params() *chaincfg.Params {
	return nil
}

func (wallet *ConfluxWallet) CurrencyCode() string {
	CurrencyCode := "CFX"
	if wallet.cfg.CoinType == wi.TestnetConflux {
		CurrencyCode = "TCFX"
	}

	return CurrencyCode
}

// ExchangeRates - return the exchangerates
func (wallet *ConfluxWallet) ExchangeRates() wi.ExchangeRates {
	return wallet.exchangeRates
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

// IsDust Check if this amount is considered dust - 10000 drip
func (wallet *ConfluxWallet) IsDust(amount big.Int) bool {
	return amount.Cmp(big.NewInt(10000)) <= 0
}

// CurrentAddress - Get the current address for the given purpose
func (wallet *ConfluxWallet) CurrentAddress(purpose wi.KeyPurpose) btcutil.Address {
	return *wallet.address
}

// NewAddress - Returns a fresh address that has never been returned by this function
func (wallet *ConfluxWallet) NewAddress(purpose wi.KeyPurpose) btcutil.Address {
	return *wallet.address
}

// DecodeAddress - Parse the address string and return an address interface
func (wallet *ConfluxWallet) DecodeAddress(addr string) (btcutil.Address, error) {
	networkID, _ := wallet.client.GetNetworkID()
	if len(addr) > 64 {
		commonAddr, err := cfxScriptToAddr(addr)
		if err != nil {
			log.Error(err.Error())
		}

		cfxAddr, err := cfxaddress.NewFromCommon(commonAddr, networkID)
		return &CfxAddress{&cfxAddr}, err
	}

	return NewCfxAddress(addr, networkID)
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

// ScriptToAddress - ?, not used
func (wallet *ConfluxWallet) ScriptToAddress(script []byte) (btcutil.Address, error) {
	return wallet.address, nil
}

// GetBalance returns the balance for the wallet
func (wallet *ConfluxWallet) GetBalance() (*big.Int, error) {
	defaultAccount, _ := wallet.client.AccountManager.GetDefault()
	balance, err := wallet.client.GetBalance(*defaultAccount, types.EpochLatestConfirmed)
	return balance.ToInt(), err
}

// GetUnconfirmedBalance returns the unconfirmed balance for the wallet
func (wallet *ConfluxWallet) GetUnconfirmedBalance() (*big.Int, error) {
	defaultAccount, _ := wallet.client.AccountManager.GetDefault()
	balance, err := wallet.client.GetBalance(*defaultAccount, types.EpochLatestState)
	return balance.ToInt(), err
}

// Balance - Get the confirmed and unconfirmed balances
func (wallet *ConfluxWallet) Balance() (confirmed, unconfirmed wi.CurrencyValue) {
	var balance, ucbalance wi.CurrencyValue
	bal, err := wallet.GetBalance()
	if err == nil {
		balance = wi.CurrencyValue{
			Value:    *bal,
			Currency: CfxCurrencyDefinition,
		}
	}
	ucbal, err := wallet.GetUnconfirmedBalance()
	ucb := big.NewInt(0)
	if err == nil {
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

// TransactionsFromEpoch - Returns a list of transactions for this wallet begining from the specified epoch
func (wallet *ConfluxWallet) TransactionsFromEpoch(startBlock *int) ([]wi.Txn, error) {
	ret := []wi.Txn{}

	// unconf, _ := wallet.db.Txns().GetAll(false)

	// txns, err := wallet.client.eClient.NormalTxByAddress(util.EnsureCorrectPrefix(wallet.account.Address().String()), startBlock, nil,
	// 	1, 0, false)
	// if err != nil && len(unconf) == 0 {
	// 	log.Error("err fetching transactions : ", err)
	// 	return []wi.Txn{}, nil
	// }

	// for _, t := range txns {
	// 	status := wi.StatusConfirmed
	// 	if t.Confirmations > 1 && t.Confirmations <= 7 {
	// 		status = wi.StatusPending
	// 	}
	// 	prefix := ""
	// 	if t.IsError != 0 {
	// 		status = wi.StatusError
	// 	}
	// 	if strings.ToLower(t.From) == strings.ToLower(wallet.address.String()) {
	// 		prefix = "-"
	// 	}

	// 	val := t.Value.Int().String()

	// 	if val == "0" { // Internal Transaction
	// 		internalTxns, err := wallet.client.eClient.InternalTxByAddress(t.To, &t.BlockNumber, &t.BlockNumber, 1, 0, false)
	// 		if err != nil && len(unconf) == 0 {
	// 			log.Errorf("Transaction Errored: %v\n", err)
	// 			continue
	// 		}
	// 		intVal, _ := new(big.Int).SetString("0", 10)
	// 		for _, v := range internalTxns {
	// 			fmt.Println(v.From, v.To, v.Value)
	// 			if v.To == t.From {
	// 				intVal = new(big.Int).Add(intVal, v.Value.Int())
	// 			}
	// 		}
	// 		val = intVal.String()
	// 	} else {
	// 		val = prefix + val
	// 	}

	// 	tnew := wi.Txn{
	// 		Txid:          util.EnsureCorrectPrefix(t.Hash),
	// 		Value:         val,
	// 		Height:        int32(t.BlockNumber),
	// 		Timestamp:     t.TimeStamp.Time(),
	// 		WatchOnly:     false,
	// 		Confirmations: int64(t.Confirmations),
	// 		Status:        wi.StatusCode(status),
	// 		Bytes:         []byte(t.Input),
	// 	}
	// 	ret = append(ret, tnew)
	// }

	// for _, u := range unconf {
	// 	u.Status = wi.StatusUnconfirmed
	// 	ret = append(ret, u)
	// }

	return ret, nil
}

// Transactions - Returns a list of transactions for this wallet
func (wallet *ConfluxWallet) Transactions() ([]wi.Txn, error) {
	return wallet.TransactionsFromEpoch(nil)
}

// GetTransaction - Get info on a specific transaction
func (wallet *ConfluxWallet) GetTransaction(txid chainhash.Hash) (wi.Txn, error) {
	tx, err := wallet.client.GetTransactionByHash(types.Hash(txid.String()))
	if err != nil {
		return wi.Txn{}, err
	}

	return wi.Txn{
		Txid:        tx.Hash.String(),
		Value:       tx.Value.String(),
		Height:      0,
		Timestamp:   time.Now(),
		WatchOnly:   false,
		Bytes:       []byte(tx.Data),
		ToAddress:   tx.To.String(),
		FromAddress: tx.From.String(),
		Outputs: []wi.TransactionOutput{
			{
				Address: CfxAddress{tx.To},
				Value:   *tx.Value.ToInt(),
				Index:   0,
			},
			{
				Address: CfxAddress{&tx.From},
				Value:   *tx.Value.ToInt(),
				Index:   1,
			},
			{
				Address: CfxAddress{tx.To},
				Value:   *tx.Value.ToInt(),
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
	} else {
		fmt.Printf("- get status result:\n%+v\n\n", status)
	}

	h, err := util.CreateChainHash(status.BestHash.String())
	if err != nil {
		log.Error(err.Error())
		h = emptyChainHash
	}
	return uint32(status.EpochNumber), *h
}

// ReSyncBlockchain - Use this to re-download merkle blocks in case of missed transactions
func (wallet *ConfluxWallet) ReSyncBlockchain(fromTime time.Time) {
	// use service here
}

// GetConfirmations - Return the number of confirmations and the height for a transaction
func (wallet *ConfluxWallet) GetConfirmations(txid chainhash.Hash) (confirms, atHeight uint32, err error) {
	tx, err := wallet.client.GetTransactionByHash(types.Hash(txid.String()))
	if err != nil {
		return 0, 0, err
	}

	highestEpoch, err := wallet.client.GetEpochNumber()

	ucfs := big.NewInt(0)
	ucfs.Sub(highestEpoch.ToInt(), tx.EpochHeight.ToInt())

	return uint32(ucfs.Uint64()), uint32(tx.EpochHeight.ToInt().Uint64()), nil
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

	return hd.NewExtendedKey(version, keyBytes, chaincode, parentFP, 0, 0, isPrivateKey), nil
}

// HasKey - Returns if the wallet has the key for the given address
func (wallet *ConfluxWallet) HasKey(addr btcutil.Address) bool {
	if !util.IsValidAddress(addr.String()) {
		return false
	}
	return wallet.address.String() == addr.String()
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
	payload = append(payload, destArr...)
	payload = append(payload, amountArr...)
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

	sig, err := crypto.Sign(txHash[:], wallet.account.privateKey)
	if err != nil {
		log.Errorf("error signing in createmultisig : %v", err)
	}
	sigs = append(sigs, wi.Signature{InputIndex: 1, Signature: sig})

	return sigs, err
}

// Multisign - Combine signatures and optionally broadcast
func (wallet *ConfluxWallet) Multisign(ins []wi.TransactionInput, outs []wi.TransactionOutput, sigs1 []wi.Signature, sigs2 []wi.Signature, redeemScript []byte, feePerByte big.Int, broadcast bool) ([]byte, error) {

	// 由于Conflux暂时未提供类似使用 abigen方式生成的go封装，导致合约不能很好调用，暂不支持多签和仲裁
	return nil, nil
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

		val := hexutil.Big(out.Value)
		gas, err := wallet.client.EstimateTxnGas(*from.address,
			*(to.(*CfxAddress).address), &val)
		if err != nil {
			return *sum
		}
		sum.Add(sum, gas.ToInt())
	}
	return *sum
}

// // Transfer will transfer the amount from this wallet to the spec address
// func (wallet *ConfluxWallet) Transfer(to string, value *big.Int, spendAll bool, fee big.Int) (common.Hash, error) {
// 	toAddress := common.HexToAddress(to)
// 	return wallet.client.Transfer(wallet.account, toAddress, value, spendAll, fee)
// }

// GenDefaultKeyStore will generate a default keystore
// func GenDefaultKeyStore(passwd string) (*Account, error) {
// 	ks := keystore.NewKeyStore("./", keystore.StandardScryptN, keystore.StandardScryptP)
// 	account, err := ks.NewAccount(passwd)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return NewAccountFromKeyfile(account.URL.Path, passwd)
// }
