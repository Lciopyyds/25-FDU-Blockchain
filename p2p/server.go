package p2p

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"mychain/core"
	"mychain/storage"
	"mychain/utils"
)

// P2PServer 表示一个节点
type P2PServer struct {
	Port    string
	BC      *core.Blockchain
	Storage *storage.FileStorage
	Peers   []string
	Mempool []core.Transaction
}

// 创建一个节点
func NewServer(port string, bc *core.Blockchain, store *storage.FileStorage) *P2PServer {
	return &P2PServer{
		Port:    port,
		BC:      bc,
		Storage: store,
		Peers:   []string{},
		Mempool: []core.Transaction{},
	}
}

// 启动 HTTP 服务器
func (s *P2PServer) Start() {
	http.HandleFunc("/latest", s.handleGetLatest)
	http.HandleFunc("/chain", s.handleGetChain)
	http.HandleFunc("/newblock", s.handleNewBlock)
	http.HandleFunc("/newtx", s.handleNewTx)
	http.HandleFunc("/mine", s.handleMine)

	addr := ":" + s.Port
	fmt.Println("节点启动 HTTP 服务，监听端口", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// ================ 下面是 4 个接口 ==================

// 返回最新区块
func (s *P2PServer) handleGetLatest(w http.ResponseWriter, r *http.Request) {
	latest := s.BC.LatestBlock()
	json.NewEncoder(w).Encode(latest)
}

// 返回整个区块链
func (s *P2PServer) handleGetChain(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.BC)
}

// 接收别人发来的新区块
func (s *P2PServer) handleNewBlock(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	var block core.Block
	if err := json.Unmarshal(body, &block); err != nil {
		fmt.Println("解析新区块失败:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	fmt.Println("收到新区块，Hash:", utils.ToHex(block.Header.Hash))

	prev := s.BC.LatestBlock()
	if prev == nil {
		fmt.Println("本地区块链为空，暂不接受该区块")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 1. 判断前一个区块哈希是否匹配本地最新区块
	if !bytes.Equal(block.Header.PreviousHash, prev.Header.Hash) {
		fmt.Println("前一个区块 Hash 不匹配，拒绝该区块")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 2. 验证 POW 是否合法
	pow := core.NewPow(&block)
	if !pow.Validate() {
		fmt.Println("POW 不合法，拒绝该区块")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. 一切正常，加入本地区块链
	s.BC.Blocks = append(s.BC.Blocks, block)
	if err := s.Storage.Save(s.BC); err != nil {
		fmt.Println("保存区块链失败:", err)
	}

	fmt.Println("成功接受并加入新区块！当前高度 =", len(s.BC.Blocks)-1)
	w.WriteHeader(http.StatusOK)
}

// /newtx：接收交易，并加入交易池，然后广播给其他节点
func (s *P2PServer) handleNewTx(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	var tx core.Transaction
	if err := json.Unmarshal(body, &tx); err != nil {
		fmt.Println("解析交易失败:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	fmt.Println("收到新交易：", tx.From, "→", tx.To, "金额", tx.Value)

	// 加入交易池
	s.Mempool = append(s.Mempool, tx)
	fmt.Println("当前交易池大小：", len(s.Mempool))

	// 判断是否是“转发来的”交易（relay=1 表示只收、不再广播）
	relayFlag := r.URL.Query().Get("relay")
	if relayFlag == "1" {
		// 来自其他节点的转发，加入本地 mempool 就行了
		w.WriteHeader(http.StatusOK)
		return
	}

	// 否则是“用户直接发给本节点”的交易，需要广播给所有邻居
	data, _ := json.Marshal(tx)
	for _, peer := range s.Peers {
		url := peer + "/newtx?relay=1" // ⭐ 关键：发给对方时带上 relay=1
		_, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			fmt.Println("广播交易到", peer, "失败：", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// 添加邻居节点
func (s *P2PServer) AddPeer(addr string) {
	fmt.Println("添加邻居节点:", addr)
	s.Peers = append(s.Peers, addr)
}

// 广播区块给所有已知节点
func (s *P2PServer) BroadcastBlock(block *core.Block) {
	for _, peer := range s.Peers {
		url := peer + "/newblock"
		fmt.Println("广播区块到", url)

		data, _ := json.Marshal(block)
		_, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			fmt.Println("广播失败:", err)
		}
	}
}

// /mine 接口：本节点挖一个新区块，并广播给所有邻居
func (s *P2PServer) handleMine(w http.ResponseWriter, r *http.Request) {
	fmt.Println("收到挖矿请求，开始挖矿...")

	// 把交易池里的所有交易拿出来（打包进区块）
	txs := make([]core.Transaction, len(s.Mempool))
	copy(txs, s.Mempool)

	// 挖矿奖励
	reward := core.Transaction{
		From:  "COINBASE",
		To:    "miner-" + s.Port,
		Value: 50,
	}
	txs = append(txs, reward)

	// 清空交易池
	s.Mempool = []core.Transaction{}

	// 使用我们之前写好的 AddBlock 挖矿并加入链
	newBlock := s.BC.AddBlock(txs)
	if err := s.Storage.Save(s.BC); err != nil {
		fmt.Println("保存区块链失败:", err)
	}

	fmt.Println("本地挖矿完成，新区块高度:", len(s.BC.Blocks)-1,
		"Hash:", utils.ToHex(newBlock.Header.Hash))

	// 广播给所有邻居
	data, _ := json.Marshal(newBlock)
	for _, peer := range s.Peers {
		url := peer + "/newblock"
		fmt.Println("广播新区块到:", url)
		_, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			fmt.Println("广播到", url, "失败:", err)
		}
	}

	fmt.Fprintf(w, "挖矿完成，高度=%d，Hash=%s\n",
		len(s.BC.Blocks)-1, utils.ToHex(newBlock.Header.Hash))
}
