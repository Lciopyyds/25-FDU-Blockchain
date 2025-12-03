package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// 对任意字节数组进行 SHA256
func Sha256(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// 将字节转为 hex 字符串，方便打印
func ToHex(data []byte) string {
	return hex.EncodeToString(data)
}
