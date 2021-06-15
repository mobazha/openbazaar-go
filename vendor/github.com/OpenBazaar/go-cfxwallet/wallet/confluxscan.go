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
	Status int32 `json:"status"` // 交易状态(0 代表成功，1 代表发生错误，当交易被跳过或未打包时为null)

	Remark    string `json:"remark"`    // 备注
	Hash      string `json:"hash"`      // 交易哈希(主键)
	From      string `json:"from"`      // 发送地址
	To        string `json:"to"`        // 接收地址
	Timestamp int64  `json:"timestamp"` // 创建时间

	Value    string `json:"value"`    // 交易额(以Drip为单位)
	GasPrice string `json:"gasPrice"` // 发送者提供的 gas 价格（以 Drip 为单位）（这笔交易每笔 gas 的价格）
	Gas      string `json:"gas"`      // 发送者提供的 gas
	GasFee   string `json:"gasFee"`

	EpochNumber uint32 `json:"epochNumber"`

	BlockHash string `json:"blockHash"`
}

type TxResult struct {
	Total     int32      `json:"total"`
	ListLimit int32      `json:"listLimit"`
	List      []TxRecord `json:"list"`
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

func (scan *ConfluxScan) getTxRecords(address string, skip int32, pageSize int32) ([]TxRecord, error) {
	url := "https://testnet.confluxscan.io/v1/transaction?skip=%d&reverse=true&limit=%d&accountAddress=%s"
	url = fmt.Sprintf(url, skip, pageSize, address)

	resp, err := scan.client.Get(url)
	if err != nil {
		return []TxRecord{}, err
	}
	decoder := json.NewDecoder(resp.Body)

	var txResult TxResult
	err = decoder.Decode(&txResult)
	if err != nil {
		return []TxRecord{}, errors.New("decode failed" + err.Error())
	}

	return txResult.List, nil
}
