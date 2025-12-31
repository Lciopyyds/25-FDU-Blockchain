# 🚀 README

# 25-FDU-Blockchain

**复旦大学区块链课程项目 · 基于 Go 的轻量级 POW 区块链原型系统**

本项目实现了一个**可运行、可挖矿、可验证签名、支持 P2P 同步的区块链系统**（无需图形化界面，命令行即可完成课程要求）。以下 README 覆盖：源码说明、编译运行、执行模式、以及与 LLM 的交流记录，满足课程提交规范。

---

## ✨ 功能概述（对照课程要求）

### 1) 数据结构

* **区块头**：`BlockHeader` 包含 `PreviousHash / Timestamp / Nonce / Hash / MerkleRoot` 等字段。
* **链式结构**：`Blockchain` 内维护 `Blocks []Block`，通过 `PreviousHash` 串联。
* **交易列表（含 coinbase）**：`Block` 内包含 `Transactions []Transaction`，挖矿时固定加入 coinbase 交易。
* **交易池**：`P2PServer.Mempool []Transaction` 维护待打包交易。

### 2) 密码与编码

* **哈希**：区块哈希使用 SHA256；交易也有独立 Hash。
* **Merkle Root**：区块头字段包含 Merkle 根，区块生成时计算。
* **公私钥 / 签名**：钱包生成 ECDSA 密钥对；交易签名在节点端验证（From 地址必须由公钥推导）。

### 3) 文件存储

* 使用 `storage/FileStorage` 将区块链存入 `data/chain_<port>.json`。
* 多节点模拟时，按端口区分文件，避免节点之间数据冲突。

### 4) 共识（POW）

* 使用 `core/pow.go` 完成工作量证明计算与验证。
* 课程要求的「无需竞争出块」通过 `/mine?addr=<address>` 手动触发。

### 5) 接收指令（启动 flag + 挖矿 + 交易）

* 节点启动：`go run ./cmd/node --port 8001 --peers http://localhost:8002,http://localhost:8003`
* 模拟挖矿：`curl -X POST "http://localhost:8001/mine?addr=<你的钱包地址>"`
* 模拟交易：`go run ./cmd/wallet send --to <地址> --value 30 --node http://localhost:8001`

### 6) 网络传输（区块同步 + 交易同步）

* `/newtx`：交易同步（广播到邻居节点，避免重复）。
* `/newblock`：新区块同步（POW + 前序哈希校验）。
* `/chain` / `/latest`：用于启动时同步最长链。

### 7) 服务器进程（多端口通信）

* HTTP 形式模拟 P2P；每个节点监听不同端口（至少 3 个节点）。
* 启动时可指定 `--peers` 来模拟邻居发现。

---

## 📦 项目结构

```
mychain/
│
├── core/               # 区块链核心：Block, Blockchain, POW, Tx
├── p2p/                # P2P 节点与 HTTP 服务
├── utils/              # 加密、地址、公钥导出等工具
├── storage/            # 区块链本地持久化
├── cmd/
│   ├── node/           # 启动节点命令
│   └── wallet/         # 轻量级钱包命令
│
├── data/               # 链文件（自动生成）
│
└── README.md
```

---

## 🧰 编译过程

```bash
# 构建节点与钱包
go build -o bin/node ./cmd/node
go build -o bin/wallet ./cmd/wallet
```

---

## 🚀 运行与执行模式

### 1. 启动 3 个节点（不同端口）

```bash
go run ./cmd/node --port 8001 --peers http://localhost:8002,http://localhost:8003
go run ./cmd/node --port 8002 --peers http://localhost:8001,http://localhost:8003
go run ./cmd/node --port 8003 --peers http://localhost:8001,http://localhost:8002
```

节点启动后会：

* 创建或加载 `data/chain_<port>.json`
* 自动调用 `/chain` 尝试同步最长链

### 2. 生成钱包

```bash
go run ./cmd/wallet gen
```

输出：

* 生成地址（即 From）
* 在当前目录写入 `wallet_priv.pem`

### 3. 发送交易（交易池）

```bash
go run ./cmd/wallet send --to <地址> --value 30 --node http://localhost:8001
```

节点会进行：

* From 地址与公钥匹配校验
* 交易签名验证
* 余额 + mempool 余额联合检查

### 4. 手动挖矿（模拟出块）

```bash
curl -X POST "http://localhost:8001/mine?addr=<你的钱包地址>"
```

* 会把 coinbase + 交易池前 N 笔交易打包
* 计算 POW，生成新区块并广播

### 5. 常用接口（调试 / 测试）

| 接口 | 说明 |
| --- | --- |
| `GET /latest` | 最新区块 |
| `GET /chain` | 整条链 |
| `POST /newtx` | 接收交易 |
| `POST /newblock` | 接收区块 |
| `POST /mine?addr=<address>` | 手动挖矿 |
| `GET /stats` | 节点统计 |
| `GET /balance?addr=<address>` | 余额查询 |

---

## 📄 代码说明（对照评分点）

### 链式结构与区块

* `core/block.go`：`Block` + `BlockHeader` 结构体，包含交易列表、Merkle 根、POW 相关字段。
* `core/blockchain.go`：`Blockchain` 维护 `Blocks`，提供 `AddBlock`、`ReplaceIfLonger` 等方法。

### 交易与交易池

* `core/transaction.go`：交易结构、哈希、签名、验证。
* `p2p/server.go`：`Mempool` 维护待打包交易；广播到邻居节点。

### POW 共识

* `core/pow.go`：目标难度 + nonce 搜索。
* `p2p/server.go`：`/mine` 手动触发出块（非竞争）。

### P2P 通信

* `p2p/server.go`：`/newtx`、`/newblock` 广播。
* `SyncWithPeers`：启动时同步最长链。

### 数据存储

* `storage/storage.go`：以 JSON 文件保存完整区块链。
* `data/chain_<port>.json`：多节点文件隔离。

---

## 🌟 独特设计点（加分项）

* **交易签名强制化**：非 coinbase 交易必须包含公钥 + 签名，否则拒绝。
* **From 地址绑定**：From = SHA256(pubKey)，拒绝伪造地址。
* **双花检测**：结合 `confirmed + pending` 余额检查。
* **多节点链同步**：最长链替换策略（ReplaceIfLonger）。

---

## 💬 与 LLM 的交流记录（节选）

> 说明：以下为实际写 README 与完善报告时的辅助对话，已整理为课堂可展示的格式。

**Q1：我需要在 README 里覆盖哪些评分点？**

**A1：** 建议按评分维度逐条覆盖：数据结构、密码与编码、文件存储、POW 共识、P2P 网络、CLI 指令、编译与执行步骤，并写清楚多节点启动方式与接口列表，便于验收。

**Q2：如何体现“交易池 + 双花检测”？**

**A2：** 描述交易进入 `Mempool` 的过程，并说明在接收新交易时结合已确认余额与待打包交易消耗量做可用余额校验，避免双花。

**Q3：能否写明多节点文件隔离方案？**

**A3：** 说明链文件以端口区分（例如 `data/chain_8001.json`），多节点模拟互不覆盖，同时保证节点启动可加载旧链。

---

## 📄 许可证

本项目用于课程学习与演示，可参考优化，但请注明引用来源。
