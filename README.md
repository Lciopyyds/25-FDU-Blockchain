# 🚀 README

# 25-FDU-Blockchain

**复旦大学区块链课程项目 · 基于 Go 的轻量级 POW 区块链原型系统**

本项目实现了一个**可运行、可挖矿、可验证签名、支持 P2P 同步并带有可视化 Dashboard**的区块链系统。

特点是——不仅能跑，而且**安全性、功能性、可视化**都明显优于普通课程实验实现。

---

## ✨ 项目功能概述

### 🧱 1. 区块链核心模块

* 区块结构（Block / BlockHeader）
* Merkle Root（可选）
* 工作量证明（Proof of Work）
* 链式结构与合法性校验
* 创世块固定（全节点一致）

### 🔐 2. 安全机制（亮点）

* **数字签名强制化（ECDSA）**
* **地址绑定（From = SHA256(pubKey)）**
* **防伪造 From（拒绝假公钥/假签名）**
* **余额系统（confirmed + pending）**
* **双花检测（交易池 + 区块链状态联合检查）**

### 🧾 3. 交易系统

* 普通交易
* Coinbase 交易（挖矿奖励）
* 交易池 Mempool
* 交易广播与去重

### ⛏ 4. 挖矿系统

* 基于 POW 的新区块产生
* 交易打包策略（普通交易 + coinbase）
* 自动计算新区块哈希
* 挖矿后自动广播

### 🌐 5. P2P 网络

* 多节点互联
* newtx / newblock 广播机制
* 节点启动自动同步 longest-chain
* ReplaceIfLonger + 链合法性检查

### 💰 6. 账户余额系统（State）

* 支持任意地址余额查询
* `/balance?addr=<address>`
* RebuildBalances 自动从链恢复状态
* 余额 + Mempool 联合检查可用资金（避免双花）

### 🧰 7. 钱包（Wallet CLI）

项目自带一个轻量级钱包 CLI，可：

```
go run ./cmd/wallet gen      # 生成密钥与地址
go run ./cmd/wallet send ... # 对交易签名并发送到节点
```

支持 ECDSA 签名，并自动推导地址。

### 🖥 8. 可视化 Dashboard（亮点界面）

访问：

```
http://localhost:8001/dashboard
```

可查看：

* 节点端口
* 区块高度
* 最新区块哈希 / Merkle Root
* 交易池大小
* Peer 列表
* **账户余额 Top10**

界面美观、结构清晰，便于展示。

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

## 🚀 快速开始

### 1. 运行节点

```bash
go run ./cmd/node --port 8001
```

### 2. 访问 Dashboard

```
http://localhost:8001/dashboard
```

### 3. 生成钱包地址

```bash
go run ./cmd/wallet gen
```

### 4. 使用钱包发起带签名的交易

```bash
go run ./cmd/wallet send --to Bob --value 30 --node http://localhost:8001
```

### 5. 手动触发挖矿

```bash
curl -X POST http://localhost:8001/mine
```

### 6. 查询余额

```
http://localhost:8001/balance?addr=<address>
```

---

## 📝 项目亮点总结（适合写在报告 / PPT）

* 完整的 POW 区块链实现
* 多节点 P2P 同步 + 最长链机制
* 交易签名强制化，杜绝伪造地址
* 余额 + Mempool 组合双花检测机制
* User-friendly Dashboard 可视化界面
* 完整的钱包命令行工具（支持签名与发送交易）
* 代码结构清晰，可读性高，易于扩展

---

## 📄 许可证

本项目用于课程学习与演示，可参考优化，但是还是不要全抄吧，引用请声明~
