package core

import (
	"crypto/ecdsa"
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
	Hash      []byte    `json:"hash"`   // 交易内容的哈希
	PubKey    []byte    `json:"pubKey"` // 发送方公钥（X.509 编码）
	Sig       []byte    `json:"sig"`    // ECDSA 签名
}

// payload 返回参与哈希 / 签名的“核心字段”字节序列
// 注意：不包含 Hash / Sig 字段本身，避免递归依赖
func (tx *Transaction) payload() []byte {
	// 这里只对 From/To/Value/Timestamp 做摘要，PubKey 也可以加入
	tmp := struct {
		From      string    `json:"from"`
		To        string    `json:"to"`
		Value     uint32    `json:"value"`
		Timestamp time.Time `json:"timestamp"`
	}{
		From:      tx.From,
		To:        tx.To,
		Value:     tx.Value,
		Timestamp: tx.Timestamp,
	}
	b, _ := json.Marshal(tmp)
	return b
}

// CalculateHash 计算交易的 Hash（对 payload 做 SHA256）
func (tx *Transaction) CalculateHash() {
	data := tx.payload()
	tx.Hash = utils.Sha256(data)
}

// Sign 使用私钥对交易签名，同时填充 PubKey、Sig、Hash
func (tx *Transaction) Sign(priv *ecdsa.PrivateKey) error {
	data := tx.payload()

	// 生成签名（ASN.1 编码的 r,s）
	sig, err := utils.SignECDSA(priv, data)
	if err != nil {
		return err
	}

	// 导出公钥
	pubBytes, err := utils.ExportPubKey(&priv.PublicKey)
	if err != nil {
		return err
	}

	tx.PubKey = pubBytes
	tx.Sig = sig
	// 最后再算一次 Hash，便于调试打印
	tx.CalculateHash()
	return nil
}

// Verify 验证交易签名是否合法
func (tx *Transaction) Verify() bool {
	if len(tx.PubKey) == 0 || len(tx.Sig) == 0 {
		// 没带公钥 / 签名，认为“未签名交易”
		return false
	}
	data := tx.payload()
	return utils.VerifyECDSA(tx.PubKey, data, tx.Sig)
}
