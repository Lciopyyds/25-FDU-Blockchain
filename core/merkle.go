package core

import "mychain/utils"

// 计算简单 Merkle Root（无补齐，直接两两拼接哈希）
func CalculateMerkleRoot(txs []Transaction) []byte {
	if len(txs) == 0 {
		return []byte{}
	}

	// 取出每个交易的 hash
	var hashes [][]byte
	for _, tx := range txs {
		hashes = append(hashes, tx.Hash)
	}

	// 如果只有一个交易，MerkleRoot = tx.Hash
	if len(hashes) == 1 {
		return hashes[0]
	}

	// 两两合并
	for len(hashes) > 1 {
		var newLevel [][]byte
		for i := 0; i < len(hashes); i += 2 {
			if i+1 == len(hashes) {
				// 奇数个时，最后一个重复一次
				combined := append(hashes[i], hashes[i]...)
				newLevel = append(newLevel, utils.Sha256(combined))
			} else {
				combined := append(hashes[i], hashes[i+1]...)
				newLevel = append(newLevel, utils.Sha256(combined))
			}
		}
		hashes = newLevel
	}

	return hashes[0]
}
