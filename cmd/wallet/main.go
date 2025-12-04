package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"mychain/core"
	"mychain/utils"
)

// 保存私钥到 PEM 文件
func savePrivKey(path string, priv *ecdsa.PrivateKey) error {
	der, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("marshal EC 私钥失败: %w", err)
	}
	block := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: der,
	}
	return os.WriteFile(path, pem.EncodeToMemory(block), 0600)
}

// 从 PEM 文件加载私钥
func loadPrivKey(path string) (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "EC PRIVATE KEY" {
		return nil, fmt.Errorf("文件不是 EC 私钥 PEM")
	}
	priv, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return priv, nil
}

// 生成密钥对并打印地址
func cmdGen() error {
	priv, pubBytes, err := utils.NewKeyPair()
	if err != nil {
		return fmt.Errorf("生成密钥对失败: %w", err)
	}

	addr := utils.PubKeyToAddress(pubBytes)
	fmt.Println("生成新的钱包：")
	fmt.Println("地址 Address :", addr)

	// 默认保存到当前目录的 wallet_priv.pem
	path := "wallet_priv.pem"
	if err := savePrivKey(path, priv); err != nil {
		return fmt.Errorf("保存私钥失败: %w", err)
	}
	fmt.Println("私钥已保存到文件:", path)
	fmt.Println("⚠ 请妥善保管该文件，丢失无法找回。")

	return nil
}

// 使用私钥文件对交易签名并发送到节点
func cmdSend() error {
	nodeURL := flag.String("node", "http://localhost:8001", "节点地址，例如 http://localhost:8001")
	skPath := flag.String("sk", "wallet_priv.pem", "私钥文件路径")
	toAddr := flag.String("to", "", "收款方地址（字符串即可）")
	value := flag.Uint("value", 0, "转账金额 (uint)")

	flag.Parse()

	if *toAddr == "" {
		return fmt.Errorf("必须指定 --to 收款地址")
	}
	if *value == 0 {
		return fmt.Errorf("转账金额必须 > 0")
	}

	// 1. 加载私钥
	priv, err := loadPrivKey(*skPath)
	if err != nil {
		return fmt.Errorf("加载私钥失败: %w", err)
	}

	// 2. 导出公钥并计算 From 地址
	pubBytes, err := utils.ExportPubKey(&priv.PublicKey)
	if err != nil {
		return fmt.Errorf("导出公钥失败: %w", err)
	}
	fromAddr := utils.PubKeyToAddress(pubBytes)

	// 3. 构造交易
	tx := core.Transaction{
		From:      fromAddr,
		To:        *toAddr,
		Value:     uint32(*value),
		Timestamp: time.Now(),
	}

	// 4. 用私钥对交易签名（会填充 PubKey、Sig、Hash）
	if err := tx.Sign(priv); err != nil {
		return fmt.Errorf("签名交易失败: %w", err)
	}

	// 5. 序列化并发送到节点 /newtx
	jsonBytes, err := jsonMarshalNoEscape(tx)
	if err != nil {
		return fmt.Errorf("序列化交易失败: %w", err)
	}

	url := *nodeURL + "/newtx"
	fmt.Println("发送交易到:", url)
	fmt.Println("From:", tx.From)
	fmt.Println("To  :", tx.To)
	fmt.Println("Value:", tx.Value)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("发送 HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("节点返回状态码:", resp.StatusCode)
	fmt.Println("节点返回内容:", string(body))

	return nil
}

// 为了避免中文等被转义，写一个简单封装
func jsonMarshalNoEscape(v interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(v)
	return buf.Bytes(), err
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法:")
		fmt.Println("  生成密钥: go run ./cmd/wallet gen")
		fmt.Println("  发送交易: go run ./cmd/wallet send --to <地址> --value <金额> [--node http://localhost:8001] [--sk wallet_priv.pem]")
		return
	}

	cmd := os.Args[1]
	// 为了方便解析子命令参数，我们手动调整 flag 包看到的 os.Args
	os.Args = append([]string{os.Args[0]}, os.Args[2:]...)

	var err error
	switch cmd {
	case "gen":
		err = cmdGen()
	case "send":
		err = cmdSend()
	default:
		fmt.Println("未知子命令:", cmd)
		fmt.Println("支持的子命令: gen, send")
		return
	}

	if err != nil {
		fmt.Println("执行命令出错:", err)
	}
}
