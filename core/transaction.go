package core

import (
	"encoding/json"
	"mychain/utils"
	"time"
)

// 非 UTXO 简化版，先用 From/To/Value 模式，后面再加“异常行为探测”
type Transaction struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Value     uint32    `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	Hash      []byte    `json:"hash"` // 之后用 SHA256 填
}

// 计算交易的 Hash（序列化后做 SHA256）
func (tx *Transaction) CalculateHash() {
	txBytes, _ := json.Marshal(tx)
	tx.Hash = utils.Sha256(txBytes)
}
