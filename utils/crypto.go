package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/asn1"
	"math/big"
)

// ecdsaSignature 用于 ASN.1 编解码 r、s
type ecdsaSignature struct {
	R, S *big.Int
}

// NewKeyPair 生成一对新的 ECDSA P-256 密钥
func NewKeyPair() (*ecdsa.PrivateKey, []byte, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	pubBytes, err := ExportPubKey(&priv.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	return priv, pubBytes, nil
}

// ExportPubKey 将公钥编码为 X.509 PKIX 格式的字节
func ExportPubKey(pub *ecdsa.PublicKey) ([]byte, error) {
	return x509.MarshalPKIXPublicKey(pub)
}

// SignECDSA 对消息做 SHA256 后使用 ECDSA 签名，返回 ASN.1 编码的 (r,s)
func SignECDSA(priv *ecdsa.PrivateKey, msg []byte) ([]byte, error) {
	hash := Sha256(msg)
	r, s, err := ecdsa.Sign(rand.Reader, priv, hash)
	if err != nil {
		return nil, err
	}
	return asn1.Marshal(ecdsaSignature{R: r, S: s})
}

// VerifyECDSA 验证 ASN.1 编码的 ECDSA 签名
func VerifyECDSA(pubBytes, msg, sig []byte) bool {
	// 解析公钥
	pubAny, err := x509.ParsePKIXPublicKey(pubBytes)
	if err != nil {
		return false
	}
	pub, ok := pubAny.(*ecdsa.PublicKey)
	if !ok {
		return false
	}

	// 解析签名
	var esig ecdsaSignature
	_, err = asn1.Unmarshal(sig, &esig)
	if err != nil {
		return false
	}

	hash := Sha256(msg)
	return ecdsa.Verify(pub, hash, esig.R, esig.S)
}
