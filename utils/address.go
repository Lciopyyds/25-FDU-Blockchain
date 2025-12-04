package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

func PubKeyToAddress(pub []byte) string {
	hash := sha256.Sum256(pub)
	return hex.EncodeToString(hash[:])
}
