package p2p

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"html"
	"mychain/core"
	"mychain/storage"
	"mychain/utils"
	"sort"
)

// 一些和挖矿相关的参数
const (
	// 每个新区块给矿工的奖励（简单整数就好）
	BlockReward = 50

	// 每个区块最多打包多少笔交易（不含 coinbase）
	MaxTxPerBlock = 5
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
	http.HandleFunc("/stats", s.handleStats)
	http.HandleFunc("/balance", s.handleBalance)
	http.HandleFunc("/dashboard", s.handleDashboard)

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

	// 收到新区块后，重算一次余额表
	s.BC.RebuildBalances()

	fmt.Println("成功接受并加入新区块！当前高度 =", len(s.BC.Blocks)-1)
	w.WriteHeader(http.StatusOK)
}

func (s *P2PServer) handleNewTx(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	var tx core.Transaction
	if err := json.Unmarshal(body, &tx); err != nil {
		fmt.Println("解析交易失败:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// ----- 1. 交易必须包含签名（除 COINBASE）-----
	if tx.From != "COINBASE" {
		if len(tx.PubKey) == 0 || len(tx.Sig) == 0 {
			fmt.Println("拒绝未签名交易：必须包含 PubKey + Sig")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing pubkey or signature"))
			return
		}
	}

	// ----- 2. 校验 From 地址是否由 PubKey 推导 -----
	if tx.From != "COINBASE" {
		expectedAddr := utils.PubKeyToAddress(tx.PubKey)
		if tx.From != expectedAddr {
			fmt.Printf("拒绝交易：From 地址伪造！声明为 %s，但公钥推导为 %s\n",
				tx.From, expectedAddr)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("forged from address"))
			return
		}
	}

	// ----- 3. 校验签名是否正确 -----
	if tx.From != "COINBASE" {
		if !tx.Verify() {
			fmt.Println("签名验证失败：拒绝该交易")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid signature"))
			return
		}
	}

	fmt.Println("交易签名验证通过 ✔")

	// ----- 4. 余额系统检查（你已经实现）-----
	if tx.From != "" && tx.From != "COINBASE" {
		confirmed := s.BC.GetBalance(tx.From)

		var pendingDelta int64 = 0
		for _, pending := range s.Mempool {
			amount := int64(pending.Value)
			if pending.From == tx.From {
				pendingDelta += amount
			}
		}

		available := confirmed - pendingDelta
		if int64(tx.Value) > available {
			fmt.Printf("交易余额不足：账户 %s 可用 %d，请求转出 %d\n",
				tx.From, available, tx.Value)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("balance not enough"))
			return
		}
	}

	// ----- 5. 交易入池 + 广播（你之前的逻辑保持不变）-----

	tx.CalculateHash()
	fmt.Println("收到新交易：", tx.From, "→", tx.To, "金额", tx.Value)

	s.Mempool = append(s.Mempool, tx)
	fmt.Println("当前交易池大小：", len(s.Mempool))

	relayFlag := r.URL.Query().Get("relay")
	if relayFlag == "1" {
		w.WriteHeader(http.StatusOK)
		return
	}

	data, _ := json.Marshal(tx)
	for _, peer := range s.Peers {
		url := peer + "/newtx?relay=1"
		http.Post(url, "application/json", bytes.NewBuffer(data))
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

	// 0. 从 URL 上拿矿工地址：/mine?addr=<钱包Address>
	minerAddr := r.URL.Query().Get("addr")
	if minerAddr == "" {
		// 没给地址就直接报错，避免奖励打到奇怪的字符串上
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "缺少矿工地址，请使用 /mine?addr=<你的钱包Address>")
		return
	}

	// 1. 现在「交易池为空」不再阻止挖矿，而是只打 coinbase
	if len(s.Mempool) == 0 {
		fmt.Println("当前交易池为空，本次只打包 coinbase 挖矿奖励交易")
	} else {
		fmt.Println("当前交易池大小：", len(s.Mempool))
	}

	// 2. 计算本次最多打包多少笔「普通交易」
	txCount := len(s.Mempool)
	if txCount > MaxTxPerBlock {
		txCount = MaxTxPerBlock
	}
	fmt.Println("本次将从交易池中打包", txCount, "笔交易进行挖矿")

	// 3. 构造 coinbase 奖励交易（放在第一笔）
	//    ✅ 奖励直接打给 minerAddr（钱包 Address），而不是 "miner-端口"
	reward := core.Transaction{
		From:  "COINBASE",
		To:    minerAddr,
		Value: BlockReward,
	}
	reward.CalculateHash()

	// 4. 组装本次要打包进区块的交易列表：
	//    [coinbase] + [前 txCount 笔普通交易]（txCount 可能为 0）
	txs := make([]core.Transaction, 0, txCount+1)
	txs = append(txs, reward)

	if txCount > 0 {
		txs = append(txs, s.Mempool[:txCount]...)
		// 5. 将已打包的普通交易从 mempool 中移除，保留未打包部分
		if txCount == len(s.Mempool) {
			// 全部打完，清空交易池
			s.Mempool = []core.Transaction{}
		} else {
			// 只打包了前 txCount 笔，后面的留在池子里
			s.Mempool = s.Mempool[txCount:]
		}
	} else {
		// 完全空池：只有 coinbase，此时直接把 mempool 清空即可（本来也为空）
		s.Mempool = []core.Transaction{}
	}

	fmt.Println("挖矿后交易池剩余：", len(s.Mempool))

	// 6. 使用 AddBlock 挖矿并加入链
	newBlock := s.BC.AddBlock(txs)
	if err := s.Storage.Save(s.BC); err != nil {
		fmt.Println("保存区块链失败:", err)
	}

	// 基于新区块刷新一次余额表
	s.BC.RebuildBalances()

	fmt.Println("本地挖矿完成，新区块高度:", len(s.BC.Blocks)-1,
		"Hash:", utils.ToHex(newBlock.Header.Hash))

	// 7. 广播给所有邻居
	data, _ := json.Marshal(newBlock)
	for _, peer := range s.Peers {
		url := peer + "/newblock"
		fmt.Println("广播新区块到:", url)
		_, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			fmt.Println("广播到", url, "失败:", err)
		}
	}

	fmt.Fprintf(w, "挖矿完成，高度=%d，Hash=%s，本次打包交易数=%d（含1笔coinbase），剩余交易池=%d\n",
		len(s.BC.Blocks)-1, utils.ToHex(newBlock.Header.Hash),
		len(txs), len(s.Mempool))
}

// fetchChainFromPeer 向某个 peer 的 /chain 接口拉取整条区块链
func (s *P2PServer) fetchChainFromPeer(peer string) ([]core.Block, error) {
	url := peer + "/chain"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status not OK: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 你的 /chain 现在返回的是 { "blocks": [...] } 还是直接 []Block？
	// 假设是 { "blocks": [...] } 形式：
	var wrapper struct {
		Blocks []core.Block `json:"blocks"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Blocks, nil
}

// SyncWithPeers 启动时向所有邻居请求它们的链，
// 如果某个邻居的链更长且合法，就替换本地链。
func (s *P2PServer) SyncWithPeers() {
	if len(s.Peers) == 0 {
		fmt.Println("[sync] 当前没有配置任何 peer，跳过同步")
		return
	}

	replaced := false

	for _, peer := range s.Peers {
		fmt.Println("[sync] 尝试从", peer, "同步区块链...")

		blocks, err := s.fetchChainFromPeer(peer)
		if err != nil {
			fmt.Println("[sync] 从", peer, "获取链失败：", err)
			continue
		}

		if len(blocks) == 0 {
			fmt.Println("[sync] 从", peer, "拿到空链，跳过")
			continue
		}

		if s.BC.ReplaceIfLonger(blocks) {
			fmt.Println("[sync] 使用", peer, "的链替换本地区块链，当前高度：", len(s.BC.Blocks))
			// 保存到本地文件
			if err := s.Storage.Save(s.BC); err != nil {
				fmt.Println("[sync] 保存链到本地失败：", err)
			}
			replaced = true
		} else {
			fmt.Println("[sync] ", peer, "的链不比本地更长或不合法，保持当前链")
		}
	}

	if !replaced {
		fmt.Println("[sync] 没有发现更长的合法链，本地链保持不变，高度：", len(s.BC.Blocks))
	}
}

// /stats：返回当前节点的一些状态信息（高度、mempool 大小、最新区块等）
func (s *P2PServer) handleStats(w http.ResponseWriter, r *http.Request) {
	// 方便前端/测试工具使用 JSON
	w.Header().Set("Content-Type", "application/json")

	height := len(s.BC.Blocks) - 1
	mempoolSize := len(s.Mempool)

	var latestHash string
	var latestMerkle string

	if height >= 0 {
		last := s.BC.LatestBlock()
		if last != nil {
			latestHash = utils.ToHex(last.Header.Hash)
			latestMerkle = utils.ToHex(last.Header.MerkleRoot)
		}
	}

	// 简单结构体作为返回体
	resp := struct {
		Port         string   `json:"port"`
		Height       int      `json:"height"`       // 当前链高度（创世块为 0）
		BlockCount   int      `json:"blockCount"`   // 区块总数
		MempoolSize  int      `json:"mempoolSize"`  // 交易池中待打包交易数量
		PeerCount    int      `json:"peerCount"`    // 已连接邻居数
		Peers        []string `json:"peers"`        // 邻居列表
		LatestHash   string   `json:"latestHash"`   // 最新区块哈希
		LatestMerkle string   `json:"latestMerkle"` // 最新区块 Merkle 根
	}{
		Port:         s.Port,
		Height:       height,
		BlockCount:   len(s.BC.Blocks),
		MempoolSize:  mempoolSize,
		PeerCount:    len(s.Peers),
		Peers:        s.Peers,
		LatestHash:   latestHash,
		LatestMerkle: latestMerkle,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Println("编码 /stats 响应失败：", err)
	}
}

// /balance?addr=Alice  查询某个账户当前余额
func (s *P2PServer) handleBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	raw := r.URL.Query().Get("addr")
	if raw == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "missing addr parameter"}`))
		return
	}

	addr := ResolveAddress(raw) // ✅ 支持传昵称或地址

	// 调用 core 层的 GetBalance
	balance := s.BC.GetBalance(addr)

	// 找展示名
	display := DisplayName(addr)

	// 返回 JSON
	resp := struct {
		Input   string `json:"input"`   // 用户传进来的原始字符串
		Address string `json:"address"` // 实际地址
		Name    string `json:"name"`    // 昵称+缩写
		Balance int64  `json:"balance"`
	}{
		Input:   raw,
		Address: addr,
		Name:    display,
		Balance: balance,
	}

	json.NewEncoder(w).Encode(resp)
}

// topBalances 返回余额前 n 名的账户（基于当前区块链状态）
// 这里只做 demo，用 map 排序实现。
func (s *P2PServer) topBalances(n int) []struct {
	Addr    string
	Balance int64
} {
	s.BC.RebuildBalances()

	type item struct {
		addr string
		bal  int64
	}
	var items []item
	for addr, bal := range s.BC.Balances {
		if addr == "COINBASE" {
			continue
		}
		items = append(items, item{addr: addr, bal: bal})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].bal > items[j].bal
	})

	if n > len(items) {
		n = len(items)
	}

	result := make([]struct {
		Addr    string
		Balance int64
	}, n)

	for i := 0; i < n; i++ {
		result[i] = struct {
			Addr    string
			Balance int64
		}{
			Addr:    DisplayName(items[i].addr), // ✅ 这里用昵称+缩写
			Balance: items[i].bal,
		}
	}
	return result
}

// /dashboard：简单的 HTML 可视化页面
func (s *P2PServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	height := len(s.BC.Blocks) - 1
	mempoolSize := len(s.Mempool)
	peerCount := len(s.Peers)

	var latestHash, latestMerkle string
	if height >= 0 {
		if last := s.BC.LatestBlock(); last != nil {
			latestHash = utils.ToHex(last.Header.Hash)
			latestMerkle = utils.ToHex(last.Header.MerkleRoot)
		}
	}

	top := s.topBalances(10)

	// 直接用 fmt.Fprintf 输出一段简单的 HTML
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>MyChain Dashboard - 节点 %s</title>
	<style>
		body { font-family: -apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica,Arial,sans-serif; margin: 20px; background: #f5f5f5; }
		h1 { margin-bottom: 10px; }
		.card { background: #fff; border-radius: 8px; padding: 16px; margin-bottom: 16px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
		table { border-collapse: collapse; width: 100%%; }
		th, td { border-bottom: 1px solid #eee; padding: 8px 4px; text-align: left; font-size: 14px; }
		th { background: #fafafa; }
		code { background: #eee; border-radius: 4px; padding: 2px 4px; }
		.badge { display: inline-block; padding: 2px 8px; border-radius: 999px; font-size: 12px; background: #eee; margin-right: 4px; }
	</style>
</head>
<body>
	<h1>MyChain Dashboard</h1>

	<div class="card">
		<h2>节点信息</h2>
		<p><span class="badge">端口</span> <code>%s</code></p>
		<p><span class="badge">当前高度</span> %d （区块总数：%d）</p>
		<p><span class="badge">交易池大小</span> %d</p>
		<p><span class="badge">已连接邻居</span> %d</p>
		<p><span class="badge">最新区块 Hash</span> <code>%s</code></p>
		<p><span class="badge">最新 Merkle Root</span> <code>%s</code></p>
	</div>

	<div class="card">
		<h2>邻居节点</h2>
		<table>
			<tr><th>#</th><th>Peer URL</th></tr>`, html.EscapeString(s.Port), html.EscapeString(s.Port), height, len(s.BC.Blocks), mempoolSize, peerCount, html.EscapeString(latestHash), html.EscapeString(latestMerkle))

	// peers 表格
	for i, p := range s.Peers {
		fmt.Fprintf(w, `<tr><td>%d</td><td><code>%s</code></td></tr>`, i+1, html.EscapeString(p))
	}

	fmt.Fprint(w, `</table>
	</div>

	<div class="card">
		<h2>账户余额 Top 10</h2>
		<table>
			<tr><th>#</th><th>Address</th><th>Balance</th></tr>`)

	for i, item := range top {
		fmt.Fprintf(w, `<tr>
			<td>%d</td>
			<td><code>%s</code></td>
			<td>%d</td>
		</tr>`, i+1, html.EscapeString(item.Addr), item.Balance)
	}

	fmt.Fprint(w, `</table>
		<p style="font-size:12px;color:#777;">数据基于当前区块链状态，每次新区块加入后自动刷新。</p>
	</div>

</body>
</html>`)
}
