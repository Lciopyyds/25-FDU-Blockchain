package core

import (
	"bytes"
	"encoding/json"
	"mychain/utils"
)

// 封装 POW 所需内容
type Pow struct {
	Block      *Block
	Difficulty []byte // 例如 {0x00, 0x00}
}

// 创建 POW 实例
func NewPow(b *Block) *Pow {
	return &Pow{
		Block:      b,
		Difficulty: []byte{0x00, 0x00}, // 难度可调
	}
}

// 准备计算哈希的数据（序列化 BlockHeader + Txs）
func (pow *Pow) prepareData(nonce uint32) []byte {
	header := pow.Block.Header

	tmp := map[string]interface{}{
		"prev":   header.PreviousHash,
		"merkle": header.MerkleRoot,
		"time":   header.Timestamp.Unix(),
		"nonce":  nonce,
	}

	data, _ := json.Marshal(tmp)
	return data
}

// 核心：挖矿
func (pow *Pow) Run() ([]byte, uint32) {
	var hash []byte
	var nonce uint32 = 0

	for {
		data := pow.prepareData(nonce)
		hash = utils.Sha256(data)

		if bytes.HasPrefix(hash, pow.Difficulty) {
			return hash, nonce
		}
		nonce++
	}
}

// Validate 用来校验一个区块是否满足 POW 要求
func (pow *Pow) Validate() bool {
	header := pow.Block.Header

	// 用当前区块头里的 Nonce 重新算一遍 hash
	data := pow.prepareData(header.Nonce)
	hash := utils.Sha256(data)

	// 必须既满足难度前缀，又和区块里存的 Hash 一致
	if !bytes.HasPrefix(hash, pow.Difficulty) {
		return false
	}
	if !bytes.Equal(hash, header.Hash) {
		return false
	}
	return true
}
