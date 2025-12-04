package core

import "bytes"

// Blockchain 就是一串 Block
type Blockchain struct {
	Blocks   []Block          `json:"blocks"`
	Balances map[string]int64 `json:"-"`
}

// 新建一个只包含创世块的区块链
func NewBlockchain() *Blockchain {
	genesis := NewGenesisBlock()
	bc := &Blockchain{
		Blocks: []Block{genesis},
	}
	bc.RebuildBalances() // 基于区块重新计算一次余额表（创世块一般没有交易）
	return bc
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

// ReplaceIfLonger 在 newBlocks 更长且合法时，用它替换当前链。
// 返回值表示是否发生了替换。
func (bc *Blockchain) ReplaceIfLonger(newBlocks []Block) bool {
	// 1. 长度不够，直接拒绝
	if len(newBlocks) <= len(bc.Blocks) {
		return false
	}

	// 2. 校验新链是否有效
	if !isValidChain(newBlocks) {
		return false
	}

	// 3. 替换本地链
	bc.Blocks = newBlocks
	return true
}

// isValidChain 用于在不修改当前 bc 的前提下，验证一条区块链是否有效
// 除了检查 prevHash / POW 以外，还会模拟一份 state，防止明显的余额为负情况。
func isValidChain(blocks []Block) bool {
	if len(blocks) == 0 {
		return false
	}

	state := make(map[string]int64)

	for i := 0; i < len(blocks); i++ {
		cur := blocks[i]

		// 1. 前后区块的前驱 Hash 必须一致
		if i > 0 {
			prev := blocks[i-1]
			if !bytes.Equal(cur.Header.PreviousHash, prev.Header.Hash) {
				return false
			}
		}

		// 2. POW 验证
		pow := NewPow(&cur)
		if !pow.Validate() {
			return false
		}

		// 3. 用一个临时 state 模拟余额变化，发现余额 < 0 直接判不合法
		for _, tx := range cur.Txs {
			amount := int64(tx.Value)

			if tx.From != "" && tx.From != "COINBASE" {
				if state[tx.From] < amount {
					// 说明在这条链的执行过程中，这个账户曾经出现过“余额不足”
					return false
				}
				state[tx.From] -= amount
			}
			if tx.To != "" {
				state[tx.To] += amount
			}
		}
	}
	return true
}

// RebuildBalances 从头扫描整条链，重建账户余额表。
// 约定：
//   - 普通交易：From 账户减去 Value，To 账户加上 Value
//   - 挖矿奖励：From == "COINBASE"，只给 To 加钱，不扣任何人
func (bc *Blockchain) RebuildBalances() {
	bc.Balances = make(map[string]int64)

	for _, block := range bc.Blocks {
		for _, tx := range block.Txs {
			amount := int64(tx.Value)

			if tx.From != "" && tx.From != "COINBASE" {
				bc.Balances[tx.From] -= amount
			}
			if tx.To != "" {
				bc.Balances[tx.To] += amount
			}
		}
	}
}

// GetBalance 返回某个地址当前在链上的余额（不包含 mempool 未确认交易的影响）
func (bc *Blockchain) GetBalance(addr string) int64 {
	if bc.Balances == nil {
		bc.RebuildBalances()
	}
	return bc.Balances[addr]
}
