package core

import "time"

// 区块头
type BlockHeader struct {
	PreviousHash []byte    `json:"previousHash"`
	MerkleRoot   []byte    `json:"merkleRoot"`
	Timestamp    time.Time `json:"timestamp"`
	Hash         []byte    `json:"hash"`
	Nonce        uint32    `json:"nonce"`
}

// 区块
type Block struct {
	Header *BlockHeader  `json:"header"`
	Txs    []Transaction `json:"txs"`
}

// 创建创世块：所有节点必须生成完全相同的创世块
func NewGenesisBlock() Block {
	// 固定的时间戳（随便选一个常量）
	const genesisTime int64 = 1700000000 // 比如 2023-11 的某个时间

	header := &BlockHeader{
		PreviousHash: nil,
		MerkleRoot:   nil,
		Timestamp:    time.Unix(genesisTime, 0), // ✅ 固定时间
	}

	block := Block{
		Header: header,
		Txs:    []Transaction{},
	}

	// POW 是确定性的：同样的 header 会得到同样的 Hash 和 Nonce
	block.Mine()

	return block
}

func (b *Block) Mine() {
	pow := NewPow(b)
	hash, nonce := pow.Run()

	b.Header.Hash = hash
	b.Header.Nonce = nonce
}

func NewBlock(prevHash []byte, txs []Transaction) Block {
	// 先为每个交易计算 hash
	for i := range txs {
		txs[i].CalculateHash()
	}

	merkle := CalculateMerkleRoot(txs)

	header := &BlockHeader{
		PreviousHash: prevHash,
		MerkleRoot:   merkle,
		Timestamp:    time.Now(),
	}

	block := Block{
		Header: header,
		Txs:    txs,
	}

	block.Mine() // 开始 POW

	return block
}
