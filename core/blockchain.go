package core

import "bytes"

// Blockchain 就是一串 Block
type Blockchain struct {
	Blocks []Block `json:"blocks"`
}

// 新建一个只包含创世块的区块链
func NewBlockchain() *Blockchain {
	genesis := NewGenesisBlock()
	return &Blockchain{
		Blocks: []Block{genesis},
	}
}

// 获取最新区块
func (bc *Blockchain) LatestBlock() *Block {
	if len(bc.Blocks) == 0 {
		return nil
	}
	return &bc.Blocks[len(bc.Blocks)-1]
}

// AddBlock 使用给定的交易创建一个新区块并追加到链上
func (bc *Blockchain) AddBlock(txs []Transaction) Block {
	prev := bc.LatestBlock()
	var prevHash []byte
	if prev != nil {
		prevHash = prev.Header.Hash
	}

	newBlock := NewBlock(prevHash, txs)
	bc.Blocks = append(bc.Blocks, newBlock)
	return newBlock
}

// IsValid 检查整条链是否合法
// 1. 每个块的 PreviousHash 是否等于前一个块的 Hash
// 2. 每个块是否通过 POW 验证
func (bc *Blockchain) IsValid() bool {
	if len(bc.Blocks) == 0 {
		return true
	}

	for i := 1; i < len(bc.Blocks); i++ {
		prev := &bc.Blocks[i-1]
		curr := &bc.Blocks[i]

		// 1. 链式结构验证
		if !bytes.Equal(curr.Header.PreviousHash, prev.Header.Hash) {
			return false
		}

		// 2. POW 验证
		pow := NewPow(curr)
		if !pow.Validate() {
			return false
		}
	}

	return true
}
