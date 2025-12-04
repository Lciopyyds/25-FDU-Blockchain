package node

import (
	"fmt"
	"os"
	"path/filepath"

	"mychain/core"
	"mychain/p2p"
	stor "mychain/storage"
)

// Config 保存一个节点的启动配置
type Config struct {
	Port  string
	Peers []string
}

// Node 表示一个完整节点（包含区块链、存储、P2P 服务器）
type Node struct {
	Config  Config
	BC      *core.Blockchain
	Storage *stor.FileStorage
	Server  *p2p.P2PServer
}

// NewNode 根据配置创建并初始化节点：加载/创建区块链，构造 P2PServer
func NewNode(cfg Config) (*Node, error) {
	// 1. 统一把所有链文件放到 data/ 子目录下，按端口区分
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	chainFile := filepath.Join(dataDir, "chain_"+cfg.Port+".json")

	fs := stor.NewFileStorage(chainFile)

	// 2. 先尝试从本地文件加载已有区块链
	bc, err := fs.Load()
	if os.IsNotExist(err) {
		fmt.Println("本地没有区块链文件，创建新链...")
		bc = core.NewBlockchain()
		if err := fs.Save(bc); err != nil {
			return nil, fmt.Errorf("保存新建区块链失败: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("加载区块链失败: %w", err)
	}

	// 2.5 基于当前区块链重建一次余额表（旧文件中没有 Balances 字段也没有关系）
	bc.RebuildBalances()

	// 3. 基于当前链和存储创建 P2P 服务器
	server := p2p.NewServer(cfg.Port, bc, fs)

	// 4. 添加配置中的邻居节点
	for _, p := range cfg.Peers {
		if p != "" {
			server.AddPeer(p)
		}
	}

	// 5. 启动前先尝试和邻居同步一次“最长链”
	fmt.Println("在节点启动前，从已配置的邻居节点尝试同步区块链...")
	server.SyncWithPeers()

	// 6. 构造 Node 返回
	n := &Node{
		Config:  cfg,
		BC:      bc,
		Storage: fs,
		Server:  server,
	}
	return n, nil
}

// Start 启动本节点（其实就是启动内部的 P2P HTTP 服务）
func (n *Node) Start() {
	fmt.Println("节点启动：端口", n.Config.Port)
	fmt.Println("当前链区块数：", len(n.BC.Blocks))
	n.Server.Start()
}
