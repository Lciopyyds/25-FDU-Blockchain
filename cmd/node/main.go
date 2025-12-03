package main

import (
	"fmt"
	"os"
	"strings"

	"mychain/node"
)

func main() {
	// 简单解析命令行参数：--port 和 --peers
	args := os.Args[1:]
	var port string
	var peers []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port":
			if i+1 < len(args) {
				port = args[i+1]
				i++
			}
		case "--peers":
			if i+1 < len(args) {
				peers = strings.Split(args[i+1], ",")
				i++
			}
		}
	}

	if port == "" {
		fmt.Println("用法: go run ./cmd/node --port 8001 [--peers http://localhost:8002,http://localhost:8003]")
		return
	}

	cfg := node.Config{
		Port:  port,
		Peers: peers,
	}

	n, err := node.NewNode(cfg)
	if err != nil {
		fmt.Println("节点初始化失败:", err)
		return
	}

	// 真正启动节点（里面会启动 HTTP 服务）
	n.Start()
}
