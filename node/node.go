package node

import (
	"fmt"
	"os"

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
	chainFile := "chain_" + cfg.Port + ".json"
	fs := stor.NewFileStorage(chainFile)

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

	server := p2p.NewServer(cfg.Port, bc, fs)
	for _, p := range cfg.Peers {
		if p != "" {
			server.AddPeer(p)
		}
	}

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
