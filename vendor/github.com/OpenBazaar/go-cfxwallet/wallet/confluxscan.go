package wallet

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

type TxRecord struct {
	EpochNumber uint32 `json:"epochNumber"`

	BlockPosition    uint32 `json:"blockPosition"`
	TransactionIndex uint32 `json:"transactionIndex"`

	Status int32 `json:"status"` // 交易状态(0 代表成功，1 代表发生错误，当交易被跳过或未打包时为null)

	Hash string `json:"hash"` // 交易哈希(主键)
	From string `json:"from"` // 发送地址
	To   string `json:"to"`   // 接收地址

	Value    string `json:"value"`    // 交易额(以Drip为单位)
	GasPrice string `json:"gasPrice"` // 发送者提供的 gas 价格（以 Drip 为单位）（这笔交易每笔 gas 的价格）
	GasFee   string `json:"gasFee"`

	Timestamp int64 `json:"timestamp"` // 创建时间
}

type TxResult struct {
	Code    int32        `json:"code"`
	Message string       `json:"message"`
	Data    TxResultData `json:"data"`
}

type TxResultData struct {
	Total int32      `json:"total"`
	List  []TxRecord `json:"list"`
}

type ConfluxScan struct {
	client *http.Client
}

func NewConfluxScan() *ConfluxScan {
	tbTransport := &http.Transport{Dial: net.Dial}
	client := &http.Client{Transport: tbTransport, Timeout: time.Minute}

	return &ConfluxScan{
		client: client,
	}
}

func (scan *ConfluxScan) getTxRecords(address string, minEpochNumber *uint32, maxEpochNumber *uint32, skip int32, pageSize int32) ([]TxRecord, error) {

	epochInfo := ""
	if minEpochNumber != nil {
		epochInfo += fmt.Sprintf("&minEpochNumber=%d", *minEpochNumber)
	}
	if maxEpochNumber != nil {
		epochInfo += fmt.Sprintf("&maxEpochNumber=%d", *maxEpochNumber)
	}
	url := "https://api-testnet.confluxscan.net/account/transactions?account=%s&skip=%d&limit=%d&sort=DESC" + epochInfo
	url = fmt.Sprintf(url, address, skip, pageSize)

	resp, err := scan.client.Get(url)
	if err != nil {
		return []TxRecord{}, err
	}
	decoder := json.NewDecoder(resp.Body)

	var txResult TxResult
	err = decoder.Decode(&txResult)
	if err != nil {
		return []TxRecord{}, errors.New("Decode failed: " + err.Error())
	}

	return txResult.Data.List, nil
}

type InteralTxAction struct {
	From string `json:"from"` // 发送地址
	To   string `json:"to"`   // 接收地址

	Value    string `json:"value"`    // 交易额(以Drip为单位)
	Gas      string `json:"gas"`      // 发送者提供的 gas
	CallType string `json:"callType"` // 发送者提供的 gas
}

type InteralTxCallResult struct {
	GasLeft    string `json:"gasLeft"`
	ReturnData string `json:"returnData"`
	Outcome    string `json:"outcome"`
}

type InteralTxCall struct {
	Action              InteralTxAction     `json:"action"`
	EpochNumber         string              `json:"epochNumber"`
	EpochHash           string              `json:"epochHash"`
	BlockHash           string              `json:"blockHash"`
	TransactionHash     string              `json:"transactionHash"`
	TransactionPosition string              `json:"transactionPosition"`
	Type                string              `json:"type"`
	Result              InteralTxCallResult `json:"result"`
}

type InteralTxTree struct {
	Action              InteralTxAction `json:"action"`
	EpochNumber         string          `json:"epochNumber"`
	EpochHash           string          `json:"epochHash"`
	BlockHash           string          `json:"blockHash"`
	TransactionHash     string          `json:"transactionHash"`
	TransactionPosition string          `json:"transactionPosition"`
	Type                string          `json:"type"`
	Index               string          `json:"index"`
	Level               uint32          `json:"level"`
	Calls               []InteralTxCall `json:"calls"`
}

// https://testnet.confluxscan.io/v1/transferTree/0x672d810422eb8b6523a4a63400373a0f1f4bfbd5f1633e79bcb9c44d8cb6750e
// getInternalTxRecords get smart contract internal transaction info
func (scan *ConfluxScan) getInternalTxRecords(txid string) (InteralTxTree, error) {
	url := "https://testnet.confluxscan.io/v1/transferTree/%s"
	url = fmt.Sprintf(url, txid)

	var txTree InteralTxTree

	resp, err := scan.client.Get(url)
	if err != nil {
		return txTree, errors.New("Http call failed: " + err.Error())
	}
	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(&txTree)
	if err != nil {
		return txTree, errors.New("Decode failed: " + err.Error())
	}

	return txTree, nil
}
