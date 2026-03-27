# 闪电贷模式设计讨论记录

> 状态：工作笔记  
> 用途：记录当前讨论结论，先不直接写进正式报告

## 1. 当前采用的总体思路

这一部分不建议写成“一个统一黑盒规则”，而建议写成：

- 一个统一定义
- 三个协议特化模式

这样更稳，也更符合本项目当前聚焦的协议范围：Aave、Balancer V2、Uniswap V2。

## 2. 统一定义

建议采用“严格闪电贷 / 严格闪电交换”的工程定义：

- 在同一笔成功交易中，已知流动性提供方先释放资产。
- 借款接收者随后进入协议规定的回调或结算窗口。
- 在交易结束前，该协议的结算条件必须被满足。
- 如果结算条件不满足，整笔交易回滚。

这里不建议把统一定义写死成“归还本金和费用”，因为三类协议的结算逻辑并不完全相同：

- Aave / Balancer：通常表现为本金加费用被收回，或者满足协议允许的费用规则。
- Uniswap V2：更本质的条件是 pair 的 fee-adjusted invariant 在交易结束前重新成立。

因此，更稳的统一表述是“满足协议结算谓词（settlement predicate）”。

## 3. 检测层次

识别器不建议一步完成判断，而建议拆成两层：

### 3.1 候选识别层

目标是快速筛出可能相关的交互：

- 已知协议地址
- 已知入口函数
- 已知回调函数
- 必要的关键事件或调用痕迹

### 3.2 精确验证层

目标是确认该候选是否真正满足严格闪电贷语义：

- lender / pair 是否先转出资产
- 是否真的触发回调或进入协议规定的结算路径
- 是否在同一笔交易里满足协议要求的还款或不变量恢复条件

因此，后续结果最好明确区分：

- `candidate interactions`
- `verified interactions`
- `candidate transactions`
- `verified transactions`

## 4. 识别单位

不建议直接把 `transaction` 作为最底层识别单位。更稳的分层是：

- `interaction`：一次 lender 发起并完成的 flash-loan / flash-swap 过程
- `asset leg`：该 interaction 中某个具体资产的借出与结算
- `transaction`：包含一个或多个 interaction 的链上交易

这样设计的原因是：

- Aave `flashLoan(...)` 可以一次借多个资产
- Balancer `flashLoan(...)` 也是多 token
- 一笔交易中可能同时出现多个 provider
- Uniswap V2 一笔交易中也可能触发多个 pair

因此，推荐口径是：

- 检测核心单位使用 `interaction`
- 金额统计和资产分布使用 `asset leg`
- 最终展示时再聚合到 `transaction`

一个可直接落地的定义是：

- `verified interaction`：协议级模式完整满足
- `verified transaction`：该交易至少包含一个 `verified interaction`

## 5. 三个协议特化模式

### 5.1 Aave Flash Loan Entrypoint and Callback Pattern

候选信号：

- `Pool.flashLoan(...)`
- `Pool.flashLoanSimple(...)`
- 回调 `executeOperation(...)`

严格验证条件：

- Pool 先向 receiver 转出资产
- receiver 进入 `executeOperation(...)`
- 同一笔交易内满足 Pool 的结算条件

需要特别收紧的边界：

- `flashLoanSimple(...)` 最干净，单资产、回调固定，默认属于严格闪电贷模式
- `flashLoan(...)` 不能直接全部算作严格闪电贷
- 如果 `interestRateModes` 中存在非零值，调用可能转成开债仓，而不是同交易归还

因此，当前建议采用以下口径：

- `flashLoanSimple(...)`：纳入严格模式
- `flashLoan(...)`：只有当所有 `interestRateModes = 0` 时，才纳入严格模式
- 任何包含非零 `interestRateModes` 的情况：排除，或单独标记为 `hybrid / debt-opening`

还需注意：

- Aave `flashLoan(...)` 中 premium 可能因权限而被豁免，因此“fee > 0”不能作为必要条件
- 更稳的条件应写成“满足协议要求的 premium 规则”，允许 premium 为 `0`

### 5.2 Balancer Vault Loan-and-Callback Pattern

候选信号：

- `Vault.flashLoan(...)`
- 回调 `receiveFlashLoan(...)`

严格验证条件：

- Vault 先向 recipient 转出 token
- recipient 收到 `receiveFlashLoan(...)` 回调
- 交易结束前 Vault 的余额和费用检查通过

这一模式的优点是：

- lender 地址集中在 Vault
- 回调接口固定
- 语义清晰，误报通常较少

因此，Balancer 非常适合作为高精度基准模式。

### 5.3 Uniswap V2 Flash Swap and Invariant-Restoration Pattern

候选信号：

- 官方 Uniswap V2 pair 的 `swap(...)`
- `data` 非空

严格验证条件：

- pair 先把输出 token 转出
- pair 触发 `uniswapV2Call(...)`
- 交易最终成功
- pair 的 fee-adjusted invariant 在交易结束前重新成立

这里有两个非常重要的限制：

- 不能只看 `Swap` 事件
- 不能只看“原 token 是否原样还回”

原因是：

- 普通 swap 和 flash swap 都会产生 `Swap` 事件
- flash swap 的偿付不一定表现为“借出什么，就原样还回什么”
- Uniswap V2 更本质的条件是不变量恢复，而不是某个表面上的资金回流路径

因此，当前建议是：

- 候选层：`swap(...) + 非空 data`
- 验证层：`uniswapV2Call(...) + 成功结算 + 不变量恢复`

## 6. 官方部署与 forks 的范围控制

主分析范围建议先只覆盖官方部署，不要一开始就把所有 V2 类 forks 混进主结果。

建议口径：

- 主数据集：只统计 canonical / official deployments
- 扩展数据集：单独处理 V2-style forks

这样做的原因是：

- 主结果更容易解释
- 协议语义更稳定
- 实现复杂度更可控

对 Uniswap V2 尤其如此，因为 forks 可能在以下方面发生变化：

- callback 命名
- fee 结构
- factory / pair 地址管理
- 细节实现与官方版本不完全一致

因此，当前建议：

- 正式方法部分先写官方 Uniswap V2 factory 产生的 pair
- Sushi、Pancake 等 V2 风格 fork 作为扩展研究，不与主结果混合

## 7. 为什么不能把所有协议都塞进一个统一黑盒规则

虽然三类协议都满足“单交易内先给钱、再执行、最后结算”的高层语义，但它们在工程实现上有明显差异：

- Aave：显式 flash loan 入口，显式回调
- Balancer：集中式 Vault，显式 flash loan 入口，显式回调
- Uniswap V2：没有单独的 flashLoan 函数，而是把 flash swap 内嵌在 `swap(...)` 中

因此，统一定义可以保留，但协议级识别模式必须特化，否则误报和漏报都会明显上升。

## 8. 对数据源的现实要求

这套方案对原始数据的可见性有要求。

如果只有 `logs / receipts`，问题会比较明显：

- Aave 和 Balancer 还能做一部分高精度识别
- Uniswap V2 很难仅靠日志完成“verified”级别判断

原因在于 Uniswap V2 的关键条件依赖：

- `swap(...)` 输入参数，尤其是 `data` 是否非空
- callback 是否真的发生
- 交易内部的资金流和结算路径

因此，如果项目目标是做 `verified interactions`，更推荐使用：

- decoded input data
- internal call trace / debug trace
- transfer trace 或等价的内部调用记录

## 9. 当前建议的最终口径

如果按当前讨论结论落地，推荐写成下面这套方法口径：

- 统一定义使用“协议结算谓词”来覆盖三类协议
- 识别核心单位使用 `interaction`
- `asset leg` 用于金额与资产层面的统计
- `transaction` 用于最终汇总展示
- Aave 只把 `flashLoanSimple(...)` 和全 `interestRateModes = 0` 的 `flashLoan(...)` 纳入严格模式
- Balancer 按 `Vault.flashLoan(...) + receiveFlashLoan(...) + 余额/费用检查` 识别
- Uniswap V2 按 `swap(...) + non-empty data + uniswapV2Call(...) + invariant restoration` 识别
- 主分析只覆盖官方部署
- V2 类 forks 单独作为扩展部分处理

可以直接复用的一句总结是：

> 我们识别的是 `verified flash-loan interactions`，并将包含至少一个 `verified interaction` 的交易标记为 `flash-loan transactions`。

## 10. 官方实现参考

以下参考用于校对上述讨论结论：

- Aave Pool: <https://github.com/aave/aave-v3-core/blob/master/contracts/protocol/pool/Pool.sol>
- Aave FlashLoanLogic: <https://github.com/aave/aave-v3-core/blob/master/contracts/protocol/libraries/logic/FlashLoanLogic.sol>
- Balancer V2 FlashLoans: <https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/vault/contracts/FlashLoans.sol>
- Uniswap V2 Pair: <https://github.com/Uniswap/v2-core/blob/master/contracts/UniswapV2Pair.sol>
- Uniswap V2 Whitepaper: <https://docs.uniswap.org/whitepaper.pdf>

## 11. flashloan-scanner 架构评估记录

### 11.1 总体判断

`flashloan-scanner` 可以作为“扫链骨架”参考，但不能直接当成闪电贷识别器使用。

更准确地说：

- 它已经具备区块推进、日志抓取、原始事件落库、游标续跑这些基础能力
- 但它当前的上层处理逻辑是桥接业务专用，不是通用协议扫描器

因此，可复用的是索引层，不是现有业务层。

### 11.2 现有架构中值得复用的部分

从当前代码结构看，它的主链路是：

- 启动 `Synchronizer`
- 由 `Synchronizer` 按区块批次抓取日志并落库
- 再由 `EventProcessor` 从原始日志表读取数据并做 ABI 级事件解析

这种分层本身是合理的，适合迁移到闪电贷检测项目中。

可复用的核心能力包括：

- 区块头推进与确认深度控制
- 批量 `eth_getLogs`
- 原始区块头与日志落库
- 用数据库记录同步进度，支持断点续跑
- 多链配置的基本组织方式

### 11.3 现有架构的主要限制

当前 `flashloan-scanner` 不能直接满足闪电贷识别需求，原因主要有 4 点：

- 配置与业务对象写死为桥接合约，不适合直接扩成 Aave、Balancer、Uniswap V2 的协议注册表
- 事件处理层只会解析 `PoolManager` 和 `MessageManager` 的事件
- 原始同步层以日志为主，没有现成的 trace 级能力
- 地址过滤模式适合少量固定合约，不适合 Uniswap V2 这种需要管理大量 pair 的场景

这意味着：

- Aave / Balancer 的候选扫描可以借它的骨架实现
- Aave / Balancer 的严格验证还需要补交易输入、receipt，最好再补 trace
- Uniswap V2 flash swap 不能只靠当前的固定地址日志模式完成

### 11.4 当前结论

对 `flashloan-scanner` 的最稳妥使用方式是：

- 保留“区块同步 + 原始日志入库 + 游标推进”这一层
- 重写上层事件解析与协议识别逻辑
- 不直接复用现有 bridge-specific processor、relayer、worker、service 逻辑

一句话总结：

> `flashloan-scanner` 更适合被改造成一个通用链上索引器，而不是直接拿来当闪电贷检测器。

## 12. 基于 flashloan-scanner 的最小改造方向

### 12.1 建议保留的模块

- `synchronizer`
- `synchronizer/node`
- `database`
- 配置加载与 migrations 的基本机制

### 12.2 建议绕开或移除的模块

- `relayer`
- `worker`
- `service`
- 当前 bridge 专用的 `event/contracts/*`

### 12.3 建议新增的模块

如果后续真的基于该仓库改造，建议新增下面几类模块：

- `protocol registry`
  - 管理 Aave、Balancer、Uniswap V2 官方地址、factory、事件签名、函数 selector
- `candidate extractor`
  - 从原始 logs / tx input 中抽取候选交互
- `verifier`
  - 将候选交互进一步验证为 `verified interactions`
- `pair discovery`
  - 专门为 Uniswap V2 管理 pair 列表
- `result writer`
  - 将候选结果、验证结果、资产维度结果写入你们自己的结果表

### 12.4 建议新增的结果表

保留原始表之后，建议新增：

- `candidate_interactions`
- `verified_interactions`
- `interaction_asset_legs`
- `scanner_cursor`（如果不复用现有游标机制）

## 13. 第一版扫描器配置草案

如果按闪电贷检测目标重新设计配置，建议改成下面这种结构：

```yaml
scanner:
  loop_interval_seconds: 5
  confirmation_depth: 6
  header_buffer_size: 500
  start_block: 0

database:
  db_host: "127.0.0.1"
  db_port: 5432
  db_user: "postgres"
  db_password: "postgres"
  db_name: "flashloan_scanner"

rpc:
  ethereum_mainnet:
    chain_id: 1
    rpc_url: "https://eth-mainnet.g.alchemy.com/v2/KEY"
    trace_rpc_url: "https://eth-mainnet.g.alchemy.com/v2/KEY"
    start_block: 19000000

protocols:
  aave:
    enabled: true
    pools:
      - "0x..."

  balancer_v2:
    enabled: true
    vaults:
      - "0x..."

  uniswap_v2:
    enabled: true
    factories:
      - "0x..."
    track_pairs_from_factory: true
    pair_cache_file: "./data/uniswap_v2_pairs.json"

verification:
  enable_trace: true
  enable_receipt_check: true
  strict_mode: true
  max_blocks_per_batch: 500

output:
  store_raw_logs: true
  store_candidates: true
  store_verified: true
```

### 13.1 这版配置的设计重点

这一版配置相较于当前 `flashloan-scanner` 的关键变化是：

- 不再围绕 `PoolManager` / `MessageManager`
- 改为围绕协议注册表组织配置
- 单独为 trace 验证预留 `trace_rpc_url`
- 将扫描参数、协议地址、验证强度、输出策略明确拆开

### 13.2 当前建议

下一步如果继续推进，不应立刻写实现，而应先把下面两件事定下来：

- 新配置最终字段集合
- 新数据库 schema

只有这两层先定清楚，后面的 candidate extractor 和 verifier 才能稳定落地。

## 14. Aave V3 识别流程（第一版完整口径）

> 说明：这一节先按 Aave V3 收口。Aave V2 可以后续再单独映射，因为接口和模式并不完全相同。

### 14.1 识别目标

我们在 Aave 中要识别的不是“任何调用了 flash loan 入口的交易”，而是：

- `candidate Aave flash-loan interaction`
- `verified strict Aave flash-loan interaction`

并进一步在交易层聚合成：

- `candidate Aave flash-loan transaction`
- `verified strict Aave flash-loan transaction`

这里的“严格”含义是：

- 同一笔成功交易内借出
- 执行 Aave 规定的 callback
- 交易结束前完成协议要求的结算
- 不把“开债仓替代归还”的情况算进严格闪电贷

### 14.2 作用范围

第一版建议只覆盖：

- Ethereum mainnet 上 Aave V3 官方 Pool
- 官方部署地址由 Aave address book 维护

这一版先不混入：

- Aave V2
- 非官方部署
- 其它借贷协议对 Aave adapter 的二次封装

### 14.3 所需输入数据

要把 Aave 识别做完整，建议准备下面几类输入：

#### 必需输入

- 官方 Pool 地址列表
- 交易基本信息：`tx_hash`、`block_number`、`timestamp`、`status`
- 交易输入数据：至少能解码 `flashLoan(...)` / `flashLoanSimple(...)`
- 交易日志：至少能读到 Aave `FlashLoan` event

#### 强烈建议输入

- ERC20 `Transfer` 日志
- internal call trace / debug trace

#### 可选输入

- 交易 receipt 的 decoded logs
- reserve -> aToken 映射表

### 14.4 第 0 步：协议注册表准备

在开始扫链前，先准备 Aave V3 的协议注册表：

- `pool_address`
- `pool_addresses_provider`
- 需要时补充 `aToken` 地址映射

第一版最稳的做法是：

- 只接受官方 address book 中登记的 Pool 地址
- 不允许用户随意传一个“看起来像 Aave”的地址

### 14.5 第 1 步：候选交易收集

Aave V3 有两个相关入口：

- `flashLoan(...)`
- `flashLoanSimple(...)`

因此候选收集优先按“入口函数”做，而不是只按 event 做。

#### 14.5.1 候选规则 A：按交易输入抓

如果能解码 tx input，则：

- `to == official_pool_address`
- selector 是 `flashLoan(...)` 或 `flashLoanSimple(...)`

满足后，记为 `candidate interaction`。

#### 14.5.2 候选规则 B：按事件补抓

如果 input 解码不完整，可以用 `FlashLoan` event 作为补充候选信号：

- 日志来自官方 Pool
- event 类型为 `FlashLoan`

但只靠 event 的候选，默认精度弱于“入口函数候选”。

#### 14.5.3 interaction 粒度怎么定义

- 一次 `flashLoanSimple(...)` 调用，对应 1 个 interaction
- 一次 `flashLoan(...)` 调用，对应 1 个 multi-asset interaction
- 该 interaction 下再展开成多个 `asset leg`

### 14.6 第 2 步：候选标准化

对候选 Aave interaction，建议先抽取下面这些字段：

- `tx_hash`
- `block_number`
- `timestamp`
- `pool_address`
- `entrypoint`：`flashLoan` 或 `flashLoanSimple`
- `receiver_address`
- `initiator`
- `assets[]`
- `amounts[]`
- `interest_rate_modes[]`（仅 `flashLoan`）
- `on_behalf_of`（仅 `flashLoan`）
- `referral_code`
- `raw_params`

这一步还不做“严格闪电贷”判断，只做结构化。

### 14.7 第 3 步：strict verification for `flashLoanSimple(...)`

`flashLoanSimple(...)` 是 Aave V3 中最干净的一类模式，第一版应优先完成它。

#### 14.7.1 验证条件

一个 `flashLoanSimple(...)` interaction 要被标记为 `verified strict`，建议至少满足：

- 交易成功，没有回滚
- 调用了官方 Pool 的 `flashLoanSimple(...)`
- 同一交易中出现来自该 Pool 的 `FlashLoan` event
- `FlashLoan` event 中：
  - `target == receiver_address`
  - `asset == input.asset`
  - `amount == input.amount`
  - `interestRateMode == 0`
- 如果有 trace：
  - receiver 合约确实被回调到 `executeOperation(asset, amount, premium, initiator, params)`
- 如果有 Transfer logs 或 trace：
  - 同一交易中完成了 `amount + premium` 的结算

#### 14.7.2 结算怎么判断

这里有一个 Aave 特有细节必须写清楚：

- Aave 在内部还款时，不是简单“把钱转回 Pool 地址”
- 源码里的 `_handleFlashLoanRepayment(...)` 会从 `receiverAddress` 拉取 `amount + premium`，并把资金转到该 reserve 对应的 `aTokenAddress`

因此，工程上更稳的判断方式是：

- 优先用 trace 确认 repayment 路径
- 如果没有 trace，则检查同 tx 中是否有对应 asset 的 `Transfer`，满足：
  - `from == receiver_address`
  - `to == reserve.aTokenAddress`
  - `value >= amount + premium`

#### 14.7.3 `flashLoanSimple(...)` 的最终判定

满足上面条件后，可标记为：

- `verified_strict = true`
- `strict_reason = simple_full_repayment`

### 14.8 第 4 步：strict verification for `flashLoan(...)`

`flashLoan(...)` 比 `flashLoanSimple(...)` 更复杂，因为它支持：

- 多资产
- `interestRateModes`
- 不归还、直接开债仓
- 对授权 flash borrower 的 fee waiver

#### 14.8.1 基础验证条件

一个 `flashLoan(...)` interaction 至少要满足：

- 交易成功
- 调用了官方 Pool 的 `flashLoan(...)`
- 同一交易中出现对应的 `FlashLoan` event
- 如果有 trace：receiver 被回调到
  - `executeOperation(address[] assets, uint256[] amounts, uint256[] premiums, address initiator, bytes params)`

#### 14.8.2 leg 级验证

对 `flashLoan(...)` 中每个 asset leg，分别判断：

- `asset[i]`
- `amount[i]`
- `interestRateModes[i]`
- 对应 `FlashLoan` event 中的 `interestRateMode`
- 是否存在 repayment 或 debt-opening 语义

这一步一定要做 leg 级别，不然多资产情况会混乱。

### 14.9 第 5 步：Aave 中最关键的排除规则

#### 14.9.1 排除规则 1：`interestRateModes[i] != 0`

这是 Aave 识别里最重要的一条。

根据 Aave V3 `FlashLoanLogic`：

- 如果 `interestRateModes[i] == NONE / 0`，走正常 repayment 路径
- 如果 `interestRateModes[i] != 0`，则不会按严格闪电贷还款，而是进入 `BorrowLogic.executeBorrow(...)` 开债仓路径

所以：

- `interestRateModes[i] == 0` 才能进入严格闪电贷验证
- `interestRateModes[i] != 0` 必须排除出严格闪电贷

#### 14.9.2 排除规则 2：混合 interaction 不算 strict interaction

如果一次 `flashLoan(...)` 调用中：

- 有的 leg 是 `interestRateModes == 0`
- 有的 leg 是 `interestRateModes != 0`

建议口径是：

- interaction 级别：不标成 `verified strict interaction`
- leg 级别：可以保留哪些 leg 满足严格还款，哪些 leg 是 debt-opening
- transaction 级别：不把这笔交易算成 `verified strict flash-loan transaction`

#### 14.9.3 排除规则 3：只有 event，没有入口和 callback 证据

如果只有 `FlashLoan` event，但缺少：

- 入口函数证据
- callback 证据
- repayment / debt-opening 证据

则建议只标成：

- `candidate`

不要直接标 `verified strict`。

#### 14.9.4 排除规则 4：非官方 Pool

即使函数名和事件都匹配，只要地址不是 Aave 官方部署的 Pool，就不纳入主结果。

### 14.10 第 6 步：最终标签逻辑

#### 14.10.1 对 `flashLoanSimple(...)`

若交易成功，且 callback 与 repayment 证据齐全，则：

- `candidate = true`
- `verified = true`
- `strict = true`

#### 14.10.2 对 `flashLoan(...)`

- 如果所有 legs 的 `interestRateModes == 0`，且所有 legs 都完成 strict repayment 验证：
  - `verified = true`
  - `strict = true`
- 只要存在任意一个 `interestRateModes != 0`：
  - `strict = false`
  - 可额外标记 `contains_debt_opening = true`

### 14.11 第 7 步：推荐输出字段

对于 Aave 部分，第一版建议至少输出：

#### interaction 级字段

- `interaction_id`
- `tx_hash`
- `block_number`
- `timestamp`
- `protocol = aave_v3`
- `pool_address`
- `entrypoint`
- `receiver_address`
- `initiator`
- `on_behalf_of`
- `candidate`
- `verified`
- `strict`
- `contains_debt_opening`
- `callback_seen`
- `repayment_seen`
- `exclusion_reason`

#### asset leg 级字段

- `interaction_id`
- `leg_index`
- `asset`
- `amount`
- `premium`
- `interest_rate_mode`
- `strict_leg`
- `repaid_to_atoken`
- `opened_debt`

### 14.12 第 8 步：如果数据不完整，怎么降级

#### 只有 input + logs，没有 trace

这种情况下：

- `flashLoanSimple(...)` 仍然可以做较高精度识别
- `flashLoan(...)` 也能做一部分 strict / non-strict 区分
- 但 callback 证据会偏弱

#### 只有 logs，没有 input

这种情况下：

- 可以做 `candidate Aave flash-loan transactions`
- 不建议直接宣称 `verified strict`
- 尤其不适合精确判断 multi-asset leg 的 `interestRateModes`

### 14.13 第 9 步：第一版实现优先级

如果按工程顺序推进，建议：

1. 先实现 `flashLoanSimple(...)` 的 `candidate -> verified strict`
2. 再实现 `flashLoan(...)` 的 multi-asset 解析
3. 再补 `interestRateModes != 0` 的排除和混合 interaction 标记
4. 最后再补 trace 级验证

这样推进的原因是：

- `flashLoanSimple(...)` 最干净
- 最适合作为第一版识别器模板
- 做通以后，再把同样的框架扩到 `flashLoan(...)`

### 14.14 当前建议的一句方法描述

如果后面要写进方法部分，Aave 可以先压缩成一句：

> 对 Aave V3，我们首先基于官方 Pool 的 `flashLoan(...)` 与 `flashLoanSimple(...)` 入口识别候选交互，再结合 `FlashLoan` 事件、callback 证据以及同交易内的 repayment / debt-opening 语义进行验证；其中仅将所有 `interestRateModes = 0` 且完成同交易结算的交互记为严格闪电贷。

### 14.15 这一节用到的官方实现参考

- Aave Pool `flashLoan(...)` / `flashLoanSimple(...)`: <https://github.com/aave/aave-v3-core/blob/master/contracts/protocol/pool/Pool.sol>
- Aave `FlashLoanLogic`: <https://github.com/aave/aave-v3-core/blob/master/contracts/protocol/libraries/logic/FlashLoanLogic.sol>
- Aave `IFlashLoanReceiver`: <https://github.com/aave/aave-v3-core/blob/master/contracts/flashloan/interfaces/IFlashLoanReceiver.sol>
- Aave address book: <https://github.com/bgd-labs/aave-address-book>

## 15. Balancer V2 识别流程（第一版完整口径）

> 说明：这一节先按 Balancer V2 官方 Vault 收口，不混入 Balancer V3，也不混入其它以 Vault 为灵感实现的外部协议。

### 15.1 识别目标

在 Balancer V2 中，我们要识别的是：

- `candidate Balancer flash-loan interaction`
- `verified strict Balancer flash-loan interaction`

并进一步聚合到交易级：

- `candidate Balancer flash-loan transaction`
- `verified strict Balancer flash-loan transaction`

这里的“严格”含义与前文一致：

- 同一笔成功交易内由 Vault 放款
- recipient 执行 Balancer 规定的 callback
- 交易结束前满足 Vault 的还款与费用条件
- 否则整笔交易回滚

### 15.2 作用范围

第一版建议只覆盖：

- Ethereum mainnet 上 Balancer V2 官方 Vault
- 官方部署地址以 Balancer deployments / 官方文档为准

这一版先不混入：

- Balancer V3
- 第三方复刻 Vault 模式的协议
- 非官方部署

### 15.3 所需输入数据

要把 Balancer V2 识别做完整，建议准备下面几类输入：

#### 必需输入

- 官方 Vault 地址
- 交易基本信息：`tx_hash`、`block_number`、`timestamp`、`status`
- 交易输入数据：能解码 `Vault.flashLoan(...)`
- 交易日志：至少能读到 `FlashLoan` event

#### 强烈建议输入

- ERC20 `Transfer` 日志
- internal call trace / debug trace

#### 可选输入

- receipt 的 decoded logs
- token decimals / token metadata

### 15.4 第 0 步：协议注册表准备

在开始扫描前，先准备 Balancer V2 的协议注册表：

- `vault_address`
- Balancer 官方部署来源

第一版建议只接受官方 canonical Vault。

这一步很重要，因为 Balancer V2 的 flash loan lender 很集中，识别精度很大程度依赖“地址是否正确”。

### 15.5 第 1 步：候选交易收集

Balancer V2 的候选收集比 Aave 更直接，因为官方 Vault 暴露了显式入口：

- `flashLoan(IFlashLoanRecipient recipient, IERC20[] tokens, uint256[] amounts, bytes userData)`

#### 15.5.1 候选规则 A：按交易输入抓

如果能解码 tx input，则：

- `to == official_vault_address`
- selector 对应 `flashLoan(...)`

满足后，记为 `candidate interaction`。

#### 15.5.2 候选规则 B：按事件补抓

如果 input 解码不完整，则可用 `FlashLoan` event 作为补充候选信号：

- 日志来自官方 Vault
- event 类型为 `FlashLoan`

但同样地，只靠 event 的候选默认精度弱于“入口函数候选”。

#### 15.5.3 interaction 粒度怎么定义

- 一次 `Vault.flashLoan(...)` 调用，对应 1 个 multi-token interaction
- 该 interaction 下再按 token 展开成多个 `asset leg`

### 15.6 第 2 步：候选标准化

对候选 Balancer interaction，建议先抽取：

- `tx_hash`
- `block_number`
- `timestamp`
- `vault_address`
- `recipient_address`
- `tokens[]`
- `amounts[]`
- `user_data`
- `entrypoint = flashLoan`

这一阶段只做结构化，不做最终严格判断。

### 15.7 第 3 步：strict verification

Balancer V2 的 strict verification 相对最干净。

根据官方 `FlashLoans.sol`，核心路径是：

- Vault 先记录每个 token 的 `preLoanBalance`
- Vault 把 token 转给 recipient
- Vault 调用 `recipient.receiveFlashLoan(tokens, amounts, feeAmounts, userData)`
- 回调结束后，Vault 检查：
  - `postLoanBalance >= preLoanBalance`
  - `receivedFeeAmount >= feeAmounts[i]`
- 条件满足后 emit `FlashLoan(recipient, token, amount, receivedFeeAmount)`

#### 15.7.1 基础验证条件

一个 Balancer flash loan interaction 要被标记为 `verified strict`，建议至少满足：

- 交易成功
- 调用了官方 Vault 的 `flashLoan(...)`
- 同一交易中出现来自官方 Vault 的 `FlashLoan` event
- event 中：
  - `recipient == input.recipient`
  - `token` 与输入数组中的对应 token 一致
  - `amount` 与输入数组中的对应 amount 一致
- 如果有 trace：
  - recipient 确实被回调到 `receiveFlashLoan(tokens, amounts, feeAmounts, userData)`

#### 15.7.2 结算怎么判断

Balancer 的还款判断比 Aave 更直接，因为 lender 与结算地址都是 Vault 本身。

工程上可以用下面两种方式验证：

- 优先方式：trace 看到 recipient 在同一交易内把 token 返还给 Vault
- 降级方式：检查同 tx 中是否有对应 token 的 `Transfer`，满足：
  - `from == recipient_address`
  - `to == vault_address`
  - `value >= amount + feeAmount`

#### 15.7.3 事件与还款的关系

Balancer V2 的 `FlashLoan` event 是在完成 callback 和 post-balance/fee check 之后才 emit 的。

因此：

- 如果交易成功且 event 存在，它本身已经是较强的成功结算信号
- 但如果你们要把结果写成 `verified strict`，仍建议补充 callback 或 repayment 证据，而不是只靠 event

### 15.8 第 4 步：leg 级验证

Balancer 的 `flashLoan(...)` 支持多 token，因此建议做 leg 级输出：

- `token[i]`
- `amount[i]`
- `feeAmount[i]`
- 是否看到对应 `FlashLoan` event
- 是否看到对应 repayment

如果所有 legs 都完成验证，则 interaction 可标记为 `verified strict`。

### 15.9 第 5 步：Balancer 中的关键排除规则

#### 15.9.1 排除规则 1：非官方 Vault

只要地址不是 Balancer 官方 canonical Vault，就不纳入主结果。

#### 15.9.2 排除规则 2：只有 event，没有入口和 callback / repayment 证据

如果只有 `FlashLoan` event，但缺少：

- 入口函数证据
- callback 证据
- repayment 证据

建议只标为：

- `candidate`

不要直接抬到最严格的 `verified strict`。

#### 15.9.3 排除规则 3：普通 Vault 资金流动

Vault 还承担 swap、join、exit 等其它功能，因此不能把“和 Vault 有 ERC20 Transfer”直接算成 flash loan。

只有满足：

- `flashLoan(...)` 入口
- `FlashLoan` event
- callback / repayment 语义

才算 flash-loan interaction。

#### 15.9.4 排除规则 4：输入与事件不一致

如果 `flashLoan(...)` 输入参数中的：

- `recipient`
- `tokens[]`
- `amounts[]`

与 event 展开的结果不一致，则：

- interaction 标为异常 candidate
- 不计入 verified strict

### 15.10 第 6 步：最终标签逻辑

若交易满足：

- 调用官方 Vault `flashLoan(...)`
- 交易成功
- 出现匹配的 `FlashLoan` event
- callback 与 repayment 证据齐全

则：

- `candidate = true`
- `verified = true`
- `strict = true`

如果只满足入口或 event，但缺 callback / repayment 证据，则：

- `candidate = true`
- `verified = false`
- `strict = false`

### 15.11 第 7 步：推荐输出字段

对于 Balancer 部分，第一版建议至少输出：

#### interaction 级字段

- `interaction_id`
- `tx_hash`
- `block_number`
- `timestamp`
- `protocol = balancer_v2`
- `vault_address`
- `recipient_address`
- `entrypoint = flashLoan`
- `candidate`
- `verified`
- `strict`
- `callback_seen`
- `repayment_seen`
- `exclusion_reason`

#### asset leg 级字段

- `interaction_id`
- `leg_index`
- `token`
- `amount`
- `fee_amount`
- `flashloan_event_seen`
- `repaid_to_vault`
- `strict_leg`

### 15.12 第 8 步：如果数据不完整，怎么降级

#### 只有 input + logs，没有 trace

这种情况下：

- Balancer 仍然可以做较高精度识别
- 因为 `flashLoan(...)` 入口很明确，`FlashLoan` event 也很集中
- 只是 callback 证据会偏弱

#### 只有 logs，没有 input

这种情况下：

- 可以做 `candidate Balancer flash-loan transactions`
- 不建议直接宣称最严格的 `verified strict`

但相比 Aave 和 Uniswap V2，Balancer 在“只有 logs”的情况下仍然更容易做高精度候选识别。

### 15.13 第 9 步：第一版实现优先级

Balancer 没有像 Aave `interestRateModes` 那样复杂的分支，因此第一版建议：

1. 先实现 `flashLoan(...)` 入口识别
2. 再对照 `FlashLoan` event 做 leg 展开
3. 再补 `receiveFlashLoan(...)` callback 证据
4. 最后再补 repayment trace / Transfer 验证

### 15.14 当前建议的一句方法描述

如果后面要写进方法部分，Balancer 可以先压缩成一句：

> 对 Balancer V2，我们基于官方 Vault 的 `flashLoan(...)` 入口识别候选交互，并结合 `FlashLoan` 事件、`receiveFlashLoan(...)` 回调证据以及同交易内返还至 Vault 的还款语义进行验证；仅在上述结算条件完整满足时，才将其标记为严格闪电贷交互。

### 15.15 这一节用到的官方实现参考

- Balancer V2 `FlashLoans.sol`: <https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/vault/contracts/FlashLoans.sol>
- Balancer deployments: <https://github.com/balancer/balancer-deployments>
- Balancer docs home: <https://docs-v2.balancer.fi/>

## 16. Uniswap V2 识别流程（第一版完整口径）

> 说明：这一节只针对官方 Uniswap V2 factory 创建的 pair，以及其原生 flash swap 语义；不把 Sushi、Pancake 等 V2-style forks 混入主结果。

### 16.1 识别目标

在 Uniswap V2 中，我们要识别的不是“发生过 swap 的交易”，而是：

- `candidate Uniswap V2 flash-swap interaction`
- `verified strict Uniswap V2 flash-swap interaction`

并进一步聚合到交易级：

- `candidate Uniswap V2 flash-swap transaction`
- `verified strict Uniswap V2 flash-swap transaction`

这里的“严格”含义是：

- pair 在同一笔成功交易中先转出 token
- 由于 `data` 非空，pair 对接收者触发 callback
- callback 执行结束后，交易仍然成功
- pair 的 fee-adjusted invariant 在交易结束前重新成立

### 16.2 作用范围

第一版建议只覆盖：

- Ethereum mainnet 官方 Uniswap V2 factory 创建的 pair
- factory 与 deployment 地址以官方文档为准

这一版先不混入：

- SushiSwap、PancakeSwap 等 V2-style forks
- 非官方 factory 创建的 pair
- 其它路由器或聚合器的抽象层

### 16.3 所需输入数据

要把 Uniswap V2 flash swap 识别做完整，建议准备下面几类输入：

#### 必需输入

- 官方 factory 地址
- 官方 pair 列表，或能从 `PairCreated` 事件动态构建 pair 列表
- 交易基本信息：`tx_hash`、`block_number`、`timestamp`、`status`
- 交易输入数据：能解码 pair 的 `swap(...)`
- 交易日志：至少能读到 pair 的 `Swap` event 与相关 ERC20 `Transfer`

#### 强烈建议输入

- internal call trace / debug trace
- receipt 的 decoded logs

#### 可选输入

- pair 的 `token0` / `token1` 缓存
- `getPair(tokenA, tokenB)` 或从 factory 导出的 pair registry

### 16.4 第 0 步：协议注册表与 pair 发现

Uniswap V2 的一个关键特点是：

- lender 不是单一地址
- lender 是官方 factory 派生出的所有 pair

因此，第一版需要先解决 pair registry。

#### 16.4.1 推荐做法

- 读取官方 factory 地址
- 回放或查询官方 `PairCreated(token0, token1, pair, allPairsLength)` 事件
- 构建本地 pair registry

建议至少保存：

- `factory_address`
- `pair_address`
- `token0`
- `token1`
- `created_block`

#### 16.4.2 为什么这一步必须先做

因为后面所有 Uniswap V2 识别都依赖：

- `to` 地址是不是官方 pair
- 事件是不是来自官方 pair

如果 pair registry 不准确，误报会非常高。

### 16.5 第 1 步：候选交易收集

Uniswap V2 的 flash swap 不是独立函数，而是嵌在 pair 的：

- `swap(uint amount0Out, uint amount1Out, address to, bytes data)`

真正把普通 swap 和 flash swap 区分开的关键，不是 `Swap` event，而是：

- `data` 是否非空

根据官方实现：

- 当 `data.length == 0`，不会触发 flash swap callback
- 当 `data.length > 0`，pair 会在转出 token 后调用 `IUniswapV2Callee(to).uniswapV2Call(...)`

#### 16.5.1 候选规则 A：按交易输入抓

如果能解码 tx input，则：

- `to == official_pair_address`
- selector 对应 pair 的 `swap(...)`
- `data.length > 0`

满足后，记为 `candidate flash-swap interaction`。

#### 16.5.2 候选规则 B：按 trace 抓

如果 trace 可用，则也可以把下面情况记为 candidate：

- 某官方 pair 在 `swap(...)` 执行中，对 `to` 发起了 `uniswapV2Call(...)`

这一条是很强的候选信号。

#### 16.5.3 为什么不能用 `Swap` event 直接抓 candidate

只靠 `Swap` event` 不能区分普通 swap 和 flash swap，原因是：

- 普通 swap 也会 emit `Swap`
- flash swap 也会 emit `Swap`
- `Swap` event 不包含 `data` 是否非空的信息
- `Swap` event 也不证明 callback 一定发生

因此：

- `Swap` event 可以作为辅助验证信号
- 不应作为 flash swap 的唯一候选入口

### 16.6 第 2 步：候选标准化

对候选 Uniswap V2 flash-swap interaction，建议先抽取：

- `tx_hash`
- `block_number`
- `timestamp`
- `factory_address`
- `pair_address`
- `token0`
- `token1`
- `amount0_out`
- `amount1_out`
- `to_address`
- `data_non_empty`
- `entrypoint = swap`

这一阶段仍然不做最终 strict 判定。

### 16.7 第 3 步：strict verification

根据官方 pair 实现，Uniswap V2 `swap(...)` 的核心顺序是：

- 检查输出数量与流动性约束
- pair 先把 `amount0Out` / `amount1Out` 乐观转出
- 如果 `data.length > 0`，则调用 `uniswapV2Call(...)`
- callback 返回后，pair 读取余额变化，推导 `amount0In` / `amount1In`
- 检查 `amount0In > 0 || amount1In > 0`
- 检查 fee-adjusted invariant
- 成功后更新储备并 emit `Swap`

#### 16.7.1 基础验证条件

一个 Uniswap V2 flash-swap interaction 要被标记为 `verified strict`，建议至少满足：

- 交易成功
- 调用了官方 pair 的 `swap(...)`
- 输入中的 `data.length > 0`
- 如果有 trace：
  - pair 确实对 `to_address` 调用了 `uniswapV2Call(sender, amount0, amount1, data)`
- 同一交易中出现来自该 pair 的 `Swap` event
- pair 的结算逻辑最终通过，交易未回滚

#### 16.7.2 `Swap` event 在这里的正确用法

`Swap` event 是有用的，但它只能作为：

- 一次成功 `swap(...)` 的结果信号

它不能单独证明：

- 这是 flash swap
- callback 一定发生
- 借出的 token 被“原样还回”

因此，正确组合应该是：

- `swap(...) + non-empty data + callback 证据 + 成功结算`

而不是：

- `Swap event -> flash swap`

#### 16.7.3 结算怎么判断

Uniswap V2 与 Aave、Balancer 最大的差异在于：

- 它的核心不是“把本金和手续费打回 lender”这一表面路径
- 而是 callback 结束后，pair 的 fee-adjusted invariant 仍然成立

因此，工程上建议按下面顺序判断：

- 优先方式：trace / input + pair 成功执行 `swap(...)`，且交易成功
- 辅助方式：
  - 读取同 tx 中与该 pair 相关的 `Transfer`
  - 观察 callback 后是否有资产重新流入 pair
  - 但不要把“是否原样还回同一种 token”写成必要条件

因为可能出现：

- 借出 token0，最终用 token1 偿付
- 或借出 token1，最终用 token0 偿付

只要 pair 的结算条件成立，交易就会成功。

### 16.8 第 4 步：interaction 粒度与 leg 设计

Uniswap V2 的一次 `swap(...)` 调用，建议定义为 1 个 interaction。

该 interaction 下可拆成两个 leg 维度：

- `token0 leg`
- `token1 leg`

但与 Aave / Balancer 不同的是，这里更适合把 leg 理解为：

- `amount0Out`
- `amount1Out`
- `amount0In`
- `amount1In`

而不是简单“借出 token 列表”。

#### 16.8.1 为什么只看 `amountOut` 不够

因为 flash swap 不只是“pair 把东西借出去”，还必须看 callback 后：

- 哪一边资产回流
- 回流多少
- 最终 invariant 是否恢复

### 16.9 第 5 步：Uniswap V2 中的关键排除规则

#### 16.9.1 排除规则 1：`data.length == 0`

这是最重要的一条。

只要 `swap(...)` 的 `data` 为空，就不应标为 flash swap candidate。

#### 16.9.2 排除规则 2：非官方 pair

即使函数和事件都匹配，只要 pair 不是官方 factory 产出的 pair，就不纳入主结果。

#### 16.9.3 排除规则 3：只有 `Swap` event，没有 input / callback 证据

如果只有 `Swap` event，但看不到：

- `swap(...)` 输入
- `data` 是否非空
- `uniswapV2Call(...)` callback

则不要标为 `verified strict`。

更保守的做法是：

- 直接不纳入 candidate
- 或者单独标成 very-weak candidate，但不进入主数据集

第一版建议采用更严格的前者。

#### 16.9.4 排除规则 4：普通 router 多跳交换

一笔路由交易中可能出现很多 pair 的 `Swap` event，但这并不意味着有 flash swap。

如果缺少：

- non-empty `data`
- callback 证据

则这些多跳普通交换必须排除。

### 16.10 第 6 步：最终标签逻辑

若交易满足：

- 调用官方 pair 的 `swap(...)`
- 输入中 `data.length > 0`
- 交易成功
- 存在 `uniswapV2Call(...)` callback 证据
- 存在对应 `Swap` 成功结算信号

则：

- `candidate = true`
- `verified = true`
- `strict = true`

如果满足：

- 官方 pair `swap(...)`
- `data.length > 0`
- 但 callback 证据不完整

则可以：

- `candidate = true`
- `verified = false`
- `strict = false`

如果只有 `Swap` event，没有 `data` 证据，则：

- 不纳入第一版严格数据集

### 16.11 第 7 步：推荐输出字段

对于 Uniswap V2 部分，第一版建议至少输出：

#### interaction 级字段

- `interaction_id`
- `tx_hash`
- `block_number`
- `timestamp`
- `protocol = uniswap_v2`
- `factory_address`
- `pair_address`
- `token0`
- `token1`
- `to_address`
- `entrypoint = swap`
- `data_non_empty`
- `candidate`
- `verified`
- `strict`
- `callback_seen`
- `swap_event_seen`
- `exclusion_reason`

#### pair-balance / leg 级字段

- `interaction_id`
- `amount0_out`
- `amount1_out`
- `amount0_in`
- `amount1_in`
- `token0_reflow_seen`
- `token1_reflow_seen`
- `strict_interpretation_ok`

### 16.12 第 8 步：如果数据不完整，怎么降级

#### 有 input + logs，没有 trace

这种情况下：

- 可以做 `candidate` 识别
- 因为 `data.length > 0` 已经是非常关键的候选条件
- 但 `verified strict` 会偏弱，因为 callback 是否真的发生只能间接推断

#### 只有 logs，没有 input

这种情况下：

- 不建议做 Uniswap V2 flash swap 主数据集
- 因为你看不到 `data` 是否非空
- 也无法把普通 swap 与 flash swap 干净区分

这是三种协议里，对数据完整性要求最高的一类。

### 16.13 第 9 步：第一版实现优先级

Uniswap V2 建议按下面顺序推进：

1. 先完成官方 factory -> pair registry
2. 再解析 pair `swap(...)` 输入，筛 `data.length > 0`
3. 再用 `Swap` event 对照成功结果
4. 再补 `uniswapV2Call(...)` callback trace 验证
5. 最后再补 pair 余额流与 invariant 恢复的辅助检查

### 16.14 当前建议的一句方法描述

如果后面要写进方法部分，Uniswap V2 可以先压缩成一句：

> 对 Uniswap V2，我们首先在官方 factory 派生的 pair 上识别 `swap(...)` 且 `data` 非空的候选交互，再结合 `uniswapV2Call(...)` 回调证据与成功结算信号进行验证；由于普通 swap 同样会产生 `Swap` 事件，因此不将 `Swap` 事件单独作为 flash swap 的识别依据。

### 16.15 这一节用到的官方实现参考

- Uniswap V2 Pair contract: <https://github.com/Uniswap/v2-core/blob/master/contracts/UniswapV2Pair.sol>
- Uniswap V2 official docs: <https://docs.uniswap.org/>
- Uniswap V2 whitepaper: <https://docs.uniswap.org/whitepaper.pdf>
- Uniswap V2 deployment addresses: <https://docs.uniswap.org/contracts/v2/reference/smart-contracts/v2-deployments>

## 17. 三协议统一识别总流程（第一版）

这一节的目标是把 Aave V3、Balancer V2、Uniswap V2 三套协议特化规则，收束成一套统一的工程流程。

这一版不追求“一个黑盒规则”，而是追求：

- 统一的数据流
- 统一的标签体系
- 协议内规则特化

### 17.1 统一输入

无论协议类型如何，第一版建议统一准备下面这些输入：

#### 必需输入

- 官方协议注册表
  - Aave V3 Pool 地址
  - Balancer V2 Vault 地址
  - Uniswap V2 factory 与 pair registry
- 基础交易数据
  - `tx_hash`
  - `block_number`
  - `timestamp`
  - `status`
- 交易输入数据
  - 至少能解码协议入口函数
- 交易日志
  - 协议事件
  - ERC20 `Transfer`

#### 强烈建议输入

- internal call trace / debug trace

### 17.2 统一输出单位

建议统一使用三层单位：

- `interaction`
- `asset leg`
- `transaction`

定义如下：

- `interaction`：一次 lender / pair 发起并完成的 flash-loan / flash-swap 过程
- `asset leg`：该 interaction 中某个具体资产维度的借出与结算信息
- `transaction`：包含一个或多个 interaction 的交易

### 17.3 总流程概览

建议把整个识别器拆成 6 个阶段：

1. 准备协议注册表
2. 扫描原始链上数据
3. 生成协议级 `candidate interactions`
4. 做协议级 strict verification
5. 聚合成 transaction 级标签
6. 抽样人工验证

### 17.4 第 1 阶段：准备协议注册表

在真正扫链之前，先构造 canonical protocol registry。

#### Aave V3

- 官方 Pool 地址

#### Balancer V2

- 官方 Vault 地址

#### Uniswap V2

- 官方 factory 地址
- factory 派生出的 pair 列表

这一阶段的目标是统一回答一个问题：

- “这个地址是不是我们认可的 lender / pair”

如果这个问题没有先定清楚，后面的候选识别一定会变脏。

### 17.5 第 2 阶段：扫描原始链上数据

统一扫描逻辑建议是：

- 按区块推进
- 抓取 block headers
- 抓取相关地址上的 logs
- 读取相关交易输入
- 如有能力，再补 trace

这一阶段只做原始数据落库和结构化，不直接给最终标签。

### 17.6 第 3 阶段：生成 `candidate interactions`

这一阶段的核心原则是：

- 优先看协议入口函数
- event 作为辅助
- 不在候选层就做过多语义推断

#### Aave V3 候选

- 官方 Pool 的 `flashLoan(...)`
- 官方 Pool 的 `flashLoanSimple(...)`
- `FlashLoan` event 作为补充信号

#### Balancer V2 候选

- 官方 Vault 的 `flashLoan(...)`
- `FlashLoan` event 作为补充信号

#### Uniswap V2 候选

- 官方 pair 的 `swap(...)`
- 且 `data.length > 0`
- 不使用单独的 `Swap` event 作为候选入口

这一阶段的产物是：

- `candidate_interactions`

### 17.7 第 4 阶段：协议级 strict verification

这一阶段开始引入协议语义。

#### Aave V3 strict verification

核心检查：

- 交易成功
- `FlashLoan` event 匹配
- callback 证据
- 同交易结算证据
- 所有 `interestRateModes == 0`

只要存在 `interestRateModes != 0`，就不能算 `verified strict interaction`。

#### Balancer V2 strict verification

核心检查：

- 交易成功
- `flashLoan(...)` 入口匹配
- `FlashLoan` event 匹配
- `receiveFlashLoan(...)` callback 证据
- 同交易返还至 Vault 的结算证据

#### Uniswap V2 strict verification

核心检查：

- 交易成功
- 官方 pair `swap(...)`
- `data.length > 0`
- `uniswapV2Call(...)` callback 证据
- `Swap` 成功结算信号
- pair 结算条件最终成立

### 17.8 第 5 阶段：统一排除规则

三类协议都应统一适用下面几类排除逻辑：

#### 地址范围排除

- 不是官方协议地址 / 官方 pair

#### 证据不足排除

- 只有 event，没有入口函数证据
- 只有入口函数，没有成功结算证据
- 缺关键字段，无法完成 strict verification

#### 协议语义排除

- Aave 存在 `interestRateModes != 0`
- Uniswap V2 `data.length == 0`
- 普通 Vault / Router 资金流动被误当作闪电贷

### 17.9 第 6 阶段：聚合成 transaction 级标签

统一聚合规则建议写成：

- `candidate transaction`：该 tx 至少包含一个 `candidate interaction`
- `verified transaction`：该 tx 至少包含一个 `verified interaction`
- `verified strict transaction`：该 tx 至少包含一个 `verified strict interaction`

如果同一笔 tx 同时包含多个 interaction，则：

- interaction 级别分别记录
- transaction 级别只做聚合，不覆盖 interaction 细节

### 17.10 统一标签体系

建议全项目统一输出下面这些标签位：

#### interaction 级

- `candidate`
- `verified`
- `strict`
- `protocol`
- `entrypoint`
- `callback_seen`
- `repayment_seen`
- `exclusion_reason`

#### transaction 级

- `contains_candidate_interaction`
- `contains_verified_interaction`
- `contains_verified_strict_interaction`
- `protocol_count`
- `interaction_count`

### 17.11 数据可信度分层

为了避免后面表述过头，建议把结果再分成 3 个可信度层级：

#### Level 1: weak candidate

- 只有 event 或局部信号
- 不进入主分析结论

#### Level 2: candidate

- 入口函数证据明确
- 但 callback / repayment 证据不完整

#### Level 3: verified strict

- 入口函数、callback、结算语义都完整
- 可以进入主数据集和主分析结果

第一版正式统计时，建议：

- 主分析只使用 `verified strict`
- `candidate` 作为补充或误差上界

### 17.12 人工抽样验证建议

在自动规则完成后，建议统一做人工抽样：

- 每个协议各抽一批 `verified strict`
- 每个协议各抽一批被排除样本
- 每个协议各抽一批边界样本

人工检查时重点看：

- 是否真有 callback
- 是否真在同交易内完成结算
- 是否把普通 swap / 普通 Vault 活动误判成闪电贷

### 17.13 当前建议的一句总方法描述

如果后面要压缩成方法总述，可以先写成：

> 我们首先基于官方协议部署地址与入口函数识别候选交互，再结合协议特定的回调与结算语义进行严格验证；最终仅将通过协议级 strict verification 的交互记为 `verified strict interactions`，并将包含至少一个此类交互的交易标记为 flash-loan transactions。

### 17.14 这一节对后续工作的意义

完成这一节之后，后面的工作顺序应当是：

1. 定 schema
2. 定配置最终字段
3. 再开始实现 scanner / candidate extractor / verifier

原因是：

- 模式层已经足够清楚
- 再继续空谈协议语义收益不大
- 现在最需要的是把统一流程变成数据结构和代码结构

## 18. 第一版最小可用 Schema 设计

### 18.1 先定一个关键决定

在进入 schema 设计后，建议对前面一个较粗的想法做收口：

- 不再使用两张独立的 `candidate_interactions` / `verified_interactions` 表
- 改为一张统一的 `protocol_interactions` 主表
- 用状态字段表示该 interaction 当前处于 `candidate / verified / strict` 哪一层

这样做的原因是：

- `candidate` 和 `verified` 不是两种不同实体，而是同一个 interaction 的不同验证状态
- 如果拆成两张表，后续会出现重复主键、同步更新、状态漂移等问题
- 一张主表 + 状态字段，更适合后续实现与分析

因此，前面文档中关于分开两张结果表的表述，后续以这一节为准。

### 18.2 第一版建议保留的层次

第一版建议把数据库分成 3 层：

- 原始层：存扫描到的区块、日志、交易输入等原始信息
- 识别层：存 `interaction` 与 `asset leg`
- 汇总层：存 transaction 级聚合结果

### 18.3 原始层：在现有骨架上建议新增的表

如果基于当前 `flashloan-scanner` 骨架改造：

- 继续保留 `block_headers_*`
- 继续保留 `contract_events_*`

但还需要补两张关键表。

#### 18.3.1 `observed_transactions`

作用：补足 tx input / status / to / from，这对 Aave 和 Uniswap V2 尤其关键。

建议字段：

- `chain_id` `BIGINT`
- `tx_hash` `VARCHAR` 主键
- `block_number` `NUMERIC(78,0)`
- `tx_index` `INTEGER`
- `from_address` `VARCHAR`
- `to_address` `VARCHAR`
- `status` `SMALLINT`
- `value` `NUMERIC(78,0)`
- `input_data` `TEXT`
- `method_selector` `VARCHAR(10)`
- `gas_used` `NUMERIC(78,0)`
- `effective_gas_price` `NUMERIC(78,0)`
- `created_at` `TIMESTAMP`

建议索引：

- `(chain_id, block_number)`
- `(chain_id, to_address)`
- `(chain_id, method_selector)`

#### 18.3.2 `scanner_cursors`

作用：记录每个扫描器、每条链当前扫到哪里，便于断点续跑。

建议字段：

- `scanner_name` `VARCHAR`
- `chain_id` `BIGINT`
- `cursor_type` `VARCHAR`
- `block_number` `NUMERIC(78,0)`
- `updated_at` `TIMESTAMP`

主键建议：

- `(scanner_name, chain_id, cursor_type)`

`cursor_type` 可以取：

- `raw_logs`
- `tx_fetch`
- `candidate_extract`
- `verification`

### 18.4 协议注册层：支持表

#### 18.4.1 `protocol_addresses`

作用：统一存官方协议地址，避免只靠配置文件硬编码。

建议字段：

- `chain_id` `BIGINT`
- `protocol` `VARCHAR`
- `address_role` `VARCHAR`
- `contract_address` `VARCHAR`
- `is_official` `BOOLEAN`
- `source` `TEXT`
- `created_at` `TIMESTAMP`

主键建议：

- `(chain_id, protocol, address_role, contract_address)`

示例：

- `aave_v3 / pool`
- `balancer_v2 / vault`
- `uniswap_v2 / factory`

#### 18.4.2 `uniswap_v2_pairs`

作用：保存官方 factory 创建的 pair registry，这是 Uniswap V2 识别的前置条件。

建议字段：

- `chain_id` `BIGINT`
- `factory_address` `VARCHAR`
- `pair_address` `VARCHAR`
- `token0` `VARCHAR`
- `token1` `VARCHAR`
- `created_block` `NUMERIC(78,0)`
- `is_official` `BOOLEAN`
- `created_at` `TIMESTAMP`

主键建议：

- `(chain_id, pair_address)`

建议索引：

- `(chain_id, factory_address)`
- `(chain_id, token0)`
- `(chain_id, token1)`

### 18.5 识别层核心表 1：`protocol_interactions`

这一张表是整个项目的核心结果表。

每一行表示：

- 一次 lender / pair 发起的 flash-loan / flash-swap interaction

建议字段如下。

#### 18.5.1 主标识字段

- `interaction_id` `UUID` 主键
- `chain_id` `BIGINT`
- `tx_hash` `VARCHAR`
- `block_number` `NUMERIC(78,0)`
- `tx_index` `INTEGER`
- `interaction_ordinal` `INTEGER`
- `timestamp` `TIMESTAMP`

这里的 `interaction_ordinal` 很重要，用于区分：

- 同一笔 tx 中多个 interaction

建议唯一约束：

- `(chain_id, tx_hash, interaction_ordinal)`

#### 18.5.2 协议识别字段

- `protocol` `VARCHAR`
- `entrypoint` `VARCHAR`
- `provider_address` `VARCHAR`
- `factory_address` `VARCHAR NULL`
- `pair_address` `VARCHAR NULL`
- `receiver_address` `VARCHAR NULL`
- `callback_target` `VARCHAR NULL`
- `initiator` `VARCHAR NULL`
- `on_behalf_of` `VARCHAR NULL`

说明：

- Aave / Balancer 的 `provider_address` 分别对应 Pool / Vault
- Uniswap V2 的 `provider_address` 可直接等于 `pair_address`
- `factory_address` 主要给 Uniswap V2 用

#### 18.5.3 状态字段

- `candidate_level` `SMALLINT`
- `verified` `BOOLEAN`
- `strict` `BOOLEAN`
- `callback_seen` `BOOLEAN`
- `settlement_seen` `BOOLEAN`
- `repayment_seen` `BOOLEAN`
- `contains_debt_opening` `BOOLEAN DEFAULT FALSE`
- `data_non_empty` `BOOLEAN NULL`

推荐 `candidate_level` 定义：

- `1 = weak candidate`
- `2 = candidate`
- `3 = verified strict`

#### 18.5.4 解释字段

- `exclusion_reason` `TEXT NULL`
- `verification_notes` `TEXT NULL`
- `raw_method_selector` `VARCHAR(10) NULL`
- `source_confidence` `SMALLINT NULL`

#### 18.5.5 通用审计字段

- `created_at` `TIMESTAMP`
- `updated_at` `TIMESTAMP`

#### 18.5.6 建议索引

- `(chain_id, block_number)`
- `(chain_id, tx_hash)`
- `(protocol, strict)`
- `(protocol, candidate_level)`
- `(provider_address)`
- `(pair_address)`

### 18.6 识别层核心表 2：`interaction_asset_legs`

这一张表用来解决多资产与 pair 双边结算问题。

每一行表示：

- 一个 interaction 下的一个资产 leg

建议字段如下。

#### 18.6.1 主标识字段

- `leg_id` `UUID` 主键
- `interaction_id` `UUID` 外键
- `leg_index` `INTEGER`

建议唯一约束：

- `(interaction_id, leg_index)`

#### 18.6.2 资产字段

- `asset_address` `VARCHAR`
- `asset_role` `VARCHAR`
- `token_side` `VARCHAR NULL`

`asset_role` 示例：

- `borrowed`
- `repaid`
- `token0_flow`
- `token1_flow`

`token_side` 主要给 Uniswap V2 用，可取：

- `token0`
- `token1`

#### 18.6.3 数量字段

- `amount_out` `NUMERIC(78,0) NULL`
- `amount_in` `NUMERIC(78,0) NULL`
- `amount_borrowed` `NUMERIC(78,0) NULL`
- `amount_repaid` `NUMERIC(78,0) NULL`
- `premium_amount` `NUMERIC(78,0) NULL`
- `fee_amount` `NUMERIC(78,0) NULL`

说明：

- Aave / Balancer 更常用 `amount_borrowed`、`amount_repaid`、`premium_amount / fee_amount`
- Uniswap V2 更常用 `amount_out`、`amount_in`

#### 18.6.4 协议特化字段

- `interest_rate_mode` `SMALLINT NULL`
- `repaid_to_address` `VARCHAR NULL`
- `opened_debt` `BOOLEAN DEFAULT FALSE`
- `strict_leg` `BOOLEAN`
- `event_seen` `BOOLEAN`
- `settlement_mode` `VARCHAR NULL`

`settlement_mode` 示例：

- `full_repayment`
- `debt_opening`
- `invariant_restored`

#### 18.6.5 审计字段

- `created_at` `TIMESTAMP`
- `updated_at` `TIMESTAMP`

建议索引：

- `(interaction_id)`
- `(asset_address)`
- `(interest_rate_mode)`
- `(opened_debt)`

### 18.7 汇总层核心表：`flashloan_transactions`

这一张表用于 transaction 级聚合，不取代 interaction 表。

每一行表示：

- 一笔交易在闪电贷识别视角下的汇总结果

建议字段：

- `chain_id` `BIGINT`
- `tx_hash` `VARCHAR` 主键
- `block_number` `NUMERIC(78,0)`
- `timestamp` `TIMESTAMP`
- `contains_candidate_interaction` `BOOLEAN`
- `contains_verified_interaction` `BOOLEAN`
- `contains_verified_strict_interaction` `BOOLEAN`
- `interaction_count` `INTEGER`
- `strict_interaction_count` `INTEGER`
- `protocol_count` `INTEGER`
- `protocols` `TEXT`
- `created_at` `TIMESTAMP`
- `updated_at` `TIMESTAMP`

其中 `protocols` 第一版可直接存：

- 逗号分隔字符串

如果后续需要更规范，再改成单独关联表。

建议索引：

- `(chain_id, block_number)`
- `(contains_verified_strict_interaction)`

### 18.8 第一版最小可用表集合

如果只做第一版可运行原型，我建议最小可用表集合是：

- `observed_transactions`
- `scanner_cursors`
- `protocol_addresses`
- `uniswap_v2_pairs`
- `protocol_interactions`
- `interaction_asset_legs`
- `flashloan_transactions`

再加上当前已有的原始表：

- `block_headers_*`
- `contract_events_*`

这已经足够支撑：

- 候选提取
- strict verification
- 交易级聚合
- 后续统计分析

### 18.9 为什么这个 schema 适合当前项目

这版 schema 的好处是：

- 和你们已经定下来的 `interaction / asset leg / transaction` 三层口径一致
- 能统一覆盖 Aave、Balancer、Uniswap V2
- 不会因为拆成 candidate / verified 两张表而引入重复数据
- 对 Uniswap V2 留出了 pair registry
- 对 Aave 留出了 `interest_rate_mode` 与 `opened_debt`

### 18.10 下一步该做什么

完成 schema 之后，下一步应当立即定：

- 配置最终字段
- 每张表的 PostgreSQL DDL
- scanner 的模块接口

如果只选一个最优先事项，建议下一步直接写：

- `protocol_interactions`
- `interaction_asset_legs`
- `flashloan_transactions`

这 3 张表的正式 DDL

## 19. 三张核心结果表的 PostgreSQL DDL（第一版草案）

> 说明：这一节先给出可落地的 PostgreSQL DDL 草案。当前版本默认使用 PostgreSQL 原生 `UUID` 与 `NUMERIC(78,0)`，并假设数据库已启用 `pgcrypto` 或等价 UUID 生成功能。若后续沿用现有 `flashloan-scanner` 的 `uint256` domain，也可以把 `NUMERIC(78,0)` 替换为该自定义类型。

### 19.1 `protocol_interactions`

```sql
CREATE TABLE IF NOT EXISTS protocol_interactions (
    interaction_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chain_id BIGINT NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    block_number NUMERIC(78, 0) NOT NULL,
    tx_index INTEGER,
    interaction_ordinal INTEGER NOT NULL,
    block_timestamp TIMESTAMP NOT NULL,

    protocol VARCHAR(32) NOT NULL,
    entrypoint VARCHAR(64) NOT NULL,
    provider_address VARCHAR(42) NOT NULL,
    factory_address VARCHAR(42),
    pair_address VARCHAR(42),
    receiver_address VARCHAR(42),
    callback_target VARCHAR(42),
    initiator VARCHAR(42),
    on_behalf_of VARCHAR(42),

    candidate_level SMALLINT NOT NULL,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    strict BOOLEAN NOT NULL DEFAULT FALSE,
    callback_seen BOOLEAN NOT NULL DEFAULT FALSE,
    settlement_seen BOOLEAN NOT NULL DEFAULT FALSE,
    repayment_seen BOOLEAN NOT NULL DEFAULT FALSE,
    contains_debt_opening BOOLEAN NOT NULL DEFAULT FALSE,
    data_non_empty BOOLEAN,

    exclusion_reason TEXT,
    verification_notes TEXT,
    raw_method_selector VARCHAR(10),
    source_confidence SMALLINT,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT protocol_interactions_chain_tx_ordinal_uniq
        UNIQUE (chain_id, tx_hash, interaction_ordinal),

    CONSTRAINT protocol_interactions_candidate_level_chk
        CHECK (candidate_level IN (1, 2, 3)),

    CONSTRAINT protocol_interactions_protocol_chk
        CHECK (protocol IN ('aave_v3', 'balancer_v2', 'uniswap_v2')),

    CONSTRAINT protocol_interactions_confidence_chk
        CHECK (source_confidence IS NULL OR source_confidence BETWEEN 1 AND 3)
);

CREATE INDEX IF NOT EXISTS protocol_interactions_chain_block_idx
    ON protocol_interactions (chain_id, block_number);

CREATE INDEX IF NOT EXISTS protocol_interactions_chain_tx_idx
    ON protocol_interactions (chain_id, tx_hash);

CREATE INDEX IF NOT EXISTS protocol_interactions_protocol_strict_idx
    ON protocol_interactions (protocol, strict);

CREATE INDEX IF NOT EXISTS protocol_interactions_protocol_candidate_idx
    ON protocol_interactions (protocol, candidate_level);

CREATE INDEX IF NOT EXISTS protocol_interactions_provider_idx
    ON protocol_interactions (provider_address);

CREATE INDEX IF NOT EXISTS protocol_interactions_pair_idx
    ON protocol_interactions (pair_address);
```

### 19.2 `interaction_asset_legs`

```sql
CREATE TABLE IF NOT EXISTS interaction_asset_legs (
    leg_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interaction_id UUID NOT NULL REFERENCES protocol_interactions(interaction_id) ON DELETE CASCADE,
    leg_index INTEGER NOT NULL,

    asset_address VARCHAR(42) NOT NULL,
    asset_role VARCHAR(32) NOT NULL,
    token_side VARCHAR(16),

    amount_out NUMERIC(78, 0),
    amount_in NUMERIC(78, 0),
    amount_borrowed NUMERIC(78, 0),
    amount_repaid NUMERIC(78, 0),
    premium_amount NUMERIC(78, 0),
    fee_amount NUMERIC(78, 0),

    interest_rate_mode SMALLINT,
    repaid_to_address VARCHAR(42),
    opened_debt BOOLEAN NOT NULL DEFAULT FALSE,
    strict_leg BOOLEAN NOT NULL DEFAULT FALSE,
    event_seen BOOLEAN NOT NULL DEFAULT FALSE,
    settlement_mode VARCHAR(32),

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT interaction_asset_legs_interaction_leg_uniq
        UNIQUE (interaction_id, leg_index),

    CONSTRAINT interaction_asset_legs_asset_role_chk
        CHECK (asset_role IN ('borrowed', 'repaid', 'token0_flow', 'token1_flow')),

    CONSTRAINT interaction_asset_legs_token_side_chk
        CHECK (token_side IS NULL OR token_side IN ('token0', 'token1')),

    CONSTRAINT interaction_asset_legs_settlement_mode_chk
        CHECK (
            settlement_mode IS NULL OR
            settlement_mode IN ('full_repayment', 'debt_opening', 'invariant_restored')
        )
);

CREATE INDEX IF NOT EXISTS interaction_asset_legs_interaction_idx
    ON interaction_asset_legs (interaction_id);

CREATE INDEX IF NOT EXISTS interaction_asset_legs_asset_idx
    ON interaction_asset_legs (asset_address);

CREATE INDEX IF NOT EXISTS interaction_asset_legs_interest_mode_idx
    ON interaction_asset_legs (interest_rate_mode);

CREATE INDEX IF NOT EXISTS interaction_asset_legs_opened_debt_idx
    ON interaction_asset_legs (opened_debt);
```

### 19.3 `flashloan_transactions`

```sql
CREATE TABLE IF NOT EXISTS flashloan_transactions (
    chain_id BIGINT NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    block_number NUMERIC(78, 0) NOT NULL,
    block_timestamp TIMESTAMP NOT NULL,

    contains_candidate_interaction BOOLEAN NOT NULL DEFAULT FALSE,
    contains_verified_interaction BOOLEAN NOT NULL DEFAULT FALSE,
    contains_verified_strict_interaction BOOLEAN NOT NULL DEFAULT FALSE,

    interaction_count INTEGER NOT NULL DEFAULT 0,
    strict_interaction_count INTEGER NOT NULL DEFAULT 0,
    protocol_count INTEGER NOT NULL DEFAULT 0,
    protocols TEXT,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY (chain_id, tx_hash)
);

CREATE INDEX IF NOT EXISTS flashloan_transactions_chain_block_idx
    ON flashloan_transactions (chain_id, block_number);

CREATE INDEX IF NOT EXISTS flashloan_transactions_strict_idx
    ON flashloan_transactions (contains_verified_strict_interaction);
```

### 19.4 当前建议的实现说明

这一版 DDL 有几个刻意的设计选择：

- `protocol_interactions` 用一张主表承载 `candidate / verified / strict` 三种状态，不拆重复表
- `interaction_asset_legs` 用来统一处理 Aave / Balancer 的多资产，以及 Uniswap V2 的双边资金流
- `flashloan_transactions` 只做 transaction 级聚合，不承载协议细节

### 19.5 如果后续要接到现有 flashloan-scanner 迁移体系

如果后续确认要把这几张表放进 `flashloan-scanner/migrations`，建议再做两件事：

1. 统一 UUID 生成策略
- 现在草案使用 `gen_random_uuid()`
- 如果你们继续沿用现有 migration 风格，也可以改回 `uuid_generate_v4()`

2. 统一数值类型策略
- 现在草案使用 `NUMERIC(78,0)`
- 如果继续沿用现有 `uint256` domain，可以整体替换

### 19.6 下一步

完成这 3 张表之后，下一步最合适的是：

- 写 `observed_transactions`、`protocol_addresses`、`uniswap_v2_pairs` 的 DDL
- 然后再开始定义 scanner 模块接口

## 20. 三张支撑表的 PostgreSQL DDL（第一版草案）

### 20.1 `observed_transactions`

```sql
CREATE TABLE IF NOT EXISTS observed_transactions (
    chain_id BIGINT NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    block_number NUMERIC(78, 0) NOT NULL,
    tx_index INTEGER,
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42),
    status SMALLINT,
    value NUMERIC(78, 0),
    input_data TEXT,
    method_selector VARCHAR(10),
    gas_used NUMERIC(78, 0),
    effective_gas_price NUMERIC(78, 0),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY (chain_id, tx_hash)
);

CREATE INDEX IF NOT EXISTS observed_transactions_chain_block_idx
    ON observed_transactions (chain_id, block_number);

CREATE INDEX IF NOT EXISTS observed_transactions_chain_to_idx
    ON observed_transactions (chain_id, to_address);

CREATE INDEX IF NOT EXISTS observed_transactions_chain_selector_idx
    ON observed_transactions (chain_id, method_selector);
```

### 20.2 `protocol_addresses`

```sql
CREATE TABLE IF NOT EXISTS protocol_addresses (
    chain_id BIGINT NOT NULL,
    protocol VARCHAR(32) NOT NULL,
    address_role VARCHAR(32) NOT NULL,
    contract_address VARCHAR(42) NOT NULL,
    is_official BOOLEAN NOT NULL DEFAULT TRUE,
    source TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY (chain_id, protocol, address_role, contract_address),

    CONSTRAINT protocol_addresses_protocol_chk
        CHECK (protocol IN ('aave_v3', 'balancer_v2', 'uniswap_v2')),

    CONSTRAINT protocol_addresses_role_chk
        CHECK (address_role IN ('pool', 'vault', 'factory', 'pair'))
);

CREATE INDEX IF NOT EXISTS protocol_addresses_chain_protocol_idx
    ON protocol_addresses (chain_id, protocol);

CREATE INDEX IF NOT EXISTS protocol_addresses_address_idx
    ON protocol_addresses (contract_address);
```

### 20.3 `uniswap_v2_pairs`

```sql
CREATE TABLE IF NOT EXISTS uniswap_v2_pairs (
    chain_id BIGINT NOT NULL,
    factory_address VARCHAR(42) NOT NULL,
    pair_address VARCHAR(42) NOT NULL,
    token0 VARCHAR(42) NOT NULL,
    token1 VARCHAR(42) NOT NULL,
    created_block NUMERIC(78, 0) NOT NULL,
    is_official BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY (chain_id, pair_address)
);

CREATE INDEX IF NOT EXISTS uniswap_v2_pairs_factory_idx
    ON uniswap_v2_pairs (chain_id, factory_address);

CREATE INDEX IF NOT EXISTS uniswap_v2_pairs_token0_idx
    ON uniswap_v2_pairs (chain_id, token0);

CREATE INDEX IF NOT EXISTS uniswap_v2_pairs_token1_idx
    ON uniswap_v2_pairs (chain_id, token1);
```

### 20.4 `scanner_cursors`

虽然上一节重点是三张支撑表，但为了让 schema 真正闭环，建议把游标表也一起定下来。

```sql
CREATE TABLE IF NOT EXISTS scanner_cursors (
    scanner_name VARCHAR(64) NOT NULL,
    chain_id BIGINT NOT NULL,
    cursor_type VARCHAR(32) NOT NULL,
    block_number NUMERIC(78, 0) NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY (scanner_name, chain_id, cursor_type),

    CONSTRAINT scanner_cursors_type_chk
        CHECK (cursor_type IN ('raw_logs', 'tx_fetch', 'candidate_extract', 'verification'))
);

CREATE INDEX IF NOT EXISTS scanner_cursors_chain_idx
    ON scanner_cursors (chain_id, cursor_type);
```

### 20.5 当前建议的实现顺序

到这一节为止，第一版 schema 已经基本齐了。

如果后面开始真正写 migration，建议顺序是：

1. `protocol_addresses`
2. `uniswap_v2_pairs`
3. `observed_transactions`
4. `protocol_interactions`
5. `interaction_asset_legs`
6. `flashloan_transactions`
7. `scanner_cursors`

### 20.6 到这里 schema 是否闭环

现在已经具备：

- 原始交易层：`observed_transactions`
- 协议注册层：`protocol_addresses`、`uniswap_v2_pairs`
- 识别层：`protocol_interactions`、`interaction_asset_legs`
- 聚合层：`flashloan_transactions`
- 运行状态层：`scanner_cursors`

因此，schema 已经可以支撑下一步进入实现设计。

### 20.7 下一步

完成 schema 之后，下一步最合适的是：

- 定 scanner 的模块接口
- 定每个模块的输入输出
- 再决定是否开始在 `flashloan-scanner` 上做代码改造

## 21. Scanner 模块接口设计（第一版）

这一节的目标是回答一个工程问题：

- 如果要基于当前讨论结果开始实现，scanner 应该拆成哪些模块
- 每个模块的输入输出是什么
- 哪些模块可以复用 `flashloan-scanner` 的骨架，哪些必须重写

### 21.1 设计原则

第一版模块设计建议遵循下面 5 个原则：

- 原始数据抓取与协议识别分层
- 候选提取与严格验证分层
- transaction、interaction、asset leg 三层分层
- 协议共有流程统一，协议语义局部特化
- 先支持离线批处理，再考虑在线持续同步

### 21.2 建议的总模块图

建议把 scanner 拆成下面 8 个模块：

1. `registry`
2. `raw_indexer`
3. `tx_fetcher`
4. `candidate_extractor`
5. `interaction_verifier`
6. `tx_aggregator`
7. `cursor_manager`
8. `store`

它们的顺序关系是：

- `registry` 提供官方地址和 pair registry
- `raw_indexer` 扫区块和日志
- `tx_fetcher` 把候选相关交易补齐到 `observed_transactions`
- `candidate_extractor` 生成 `protocol_interactions` 的 candidate 行
- `interaction_verifier` 更新 `verified / strict` 与 `interaction_asset_legs`
- `tx_aggregator` 回写 `flashloan_transactions`
- `cursor_manager` 维护每个阶段的进度
- `store` 为所有模块提供统一持久化接口

### 21.3 哪些模块可以复用 `flashloan-scanner`

#### 可以复用的部分

- 区块推进逻辑
- `eth_getLogs` 批量抓取
- 原始日志落库思路
- 多链配置组织方式
- 游标 / 断点续跑思路

#### 建议重写的部分

- bridge-specific `event processor`
- `relayer`
- `worker`
- `service`
- 所有现有 bridge 业务表的上层处理

### 21.4 核心共享数据结构

在模块接口层，建议先统一几种共享结构。

#### 21.4.1 `ObservedTransaction`

```go
 type ObservedTransaction struct {
     ChainID           uint64
     TxHash            string
     BlockNumber       string
     TxIndex           uint64
     FromAddress       string
     ToAddress         string
     Status            uint8
     Value             string
     InputData         string
     MethodSelector    string
     GasUsed           string
     EffectiveGasPrice string
 }
```

#### 21.4.2 `CandidateInteraction`

```go
 type CandidateInteraction struct {
     InteractionOrdinal int
     ChainID            uint64
     TxHash             string
     BlockNumber        string
     Protocol           string
     Entrypoint         string
     ProviderAddress    string
     PairAddress        string
     ReceiverAddress    string
     CandidateLevel     int
     RawMethodSelector  string
     DataNonEmpty       *bool
 }
```

#### 21.4.3 `VerifiedInteraction`

```go
 type VerifiedInteraction struct {
     InteractionID        string
     Verified             bool
     Strict               bool
     CallbackSeen         bool
     SettlementSeen       bool
     RepaymentSeen        bool
     ContainsDebtOpening  bool
     ExclusionReason      *string
     VerificationNotes    *string
 }
```

#### 21.4.4 `InteractionLeg`

```go
 type InteractionLeg struct {
     InteractionID     string
     LegIndex          int
     AssetAddress      string
     AssetRole         string
     TokenSide         *string
     AmountOut         *string
     AmountIn          *string
     AmountBorrowed    *string
     AmountRepaid      *string
     PremiumAmount     *string
     FeeAmount         *string
     InterestRateMode  *int
     RepaidToAddress   *string
     OpenedDebt        bool
     StrictLeg         bool
     EventSeen         bool
     SettlementMode    *string
 }
```

#### 21.4.5 `TxSummary`

```go
 type TxSummary struct {
     ChainID                            uint64
     TxHash                             string
     BlockNumber                        string
     ContainsCandidateInteraction       bool
     ContainsVerifiedInteraction        bool
     ContainsVerifiedStrictInteraction  bool
     InteractionCount                   int
     StrictInteractionCount             int
     ProtocolCount                      int
     Protocols                          []string
 }
```

### 21.5 `registry` 模块

职责：

- 加载协议注册表
- 提供官方地址判断
- 提供 Uniswap V2 pair registry

建议接口：

```go
 type Registry interface {
     IsOfficialAavePool(chainID uint64, address string) bool
     IsOfficialBalancerVault(chainID uint64, address string) bool
     IsOfficialUniswapV2Factory(chainID uint64, address string) bool
     IsOfficialUniswapV2Pair(chainID uint64, address string) bool
     GetUniswapV2Pair(chainID uint64, pair string) (*UniswapV2Pair, error)
     ListTrackedAddresses(chainID uint64) []string
 }
```

说明：

- `ListTrackedAddresses` 主要给原始日志过滤用
- 但 Uniswap V2 pair 数量大，后续可能不能只靠这个函数完成全部过滤

### 21.6 `raw_indexer` 模块

职责：

- 按区块推进
- 批量抓取 headers 和 logs
- 写入原始表
- 更新 `raw_logs` cursor

建议接口：

```go
 type RawIndexer interface {
     RunOnce(ctx context.Context, chainID uint64, fromBlock, toBlock uint64) error
     RunLoop(ctx context.Context, chainID uint64) error
 }
```

更细的内部接口建议：

```go
 type HeaderSource interface {
     BlockHeadersByRange(ctx context.Context, chainID uint64, fromBlock, toBlock uint64) ([]Header, error)
 }

 type LogSource interface {
     FilterLogs(ctx context.Context, chainID uint64, fromBlock, toBlock uint64, addresses []string) ([]RawLog, error)
 }
```

### 21.7 `tx_fetcher` 模块

职责：

- 对原始日志中出现的相关交易，补抓 tx input / receipt 状态
- 写入 `observed_transactions`
- 更新 `tx_fetch` cursor

建议接口：

```go
 type TxFetcher interface {
     FetchByTxHash(ctx context.Context, chainID uint64, txHash string) (*ObservedTransaction, error)
     FetchRange(ctx context.Context, chainID uint64, fromBlock, toBlock uint64) error
 }
```

如果后续实现中发现按 block range 太重，也可以改成：

- 基于新入库日志的 tx hash 去重后逐笔补抓

### 21.8 `candidate_extractor` 模块

职责：

- 基于 `observed_transactions + contract_events + registry`
- 生成 candidate interactions
- 回写 `protocol_interactions`
- 更新 `candidate_extract` cursor

建议做成协议分发器 + 协议专用 extractor。

统一接口建议：

```go
 type CandidateExtractor interface {
     Protocol() string
     Extract(ctx context.Context, chainID uint64, txs []ObservedTransaction) ([]CandidateInteraction, []InteractionLeg, error)
 }
```

建议具体实现：

- `AaveV3CandidateExtractor`
- `BalancerV2CandidateExtractor`
- `UniswapV2CandidateExtractor`

### 21.9 `interaction_verifier` 模块

职责：

- 读取 candidate interactions
- 结合 logs、tx input、trace 做验证
- 更新 `protocol_interactions`
- 写入 / 更新 `interaction_asset_legs`
- 更新 `verification` cursor

统一接口建议：

```go
 type InteractionVerifier interface {
     Protocol() string
     Verify(ctx context.Context, chainID uint64, interaction CandidateInteraction) (*VerifiedInteraction, []InteractionLeg, error)
 }
```

建议具体实现：

- `AaveV3Verifier`
- `BalancerV2Verifier`
- `UniswapV2Verifier`

### 21.10 `tx_aggregator` 模块

职责：

- 从 `protocol_interactions` 聚合同一 tx 的多条 interaction
- 回写 `flashloan_transactions`

建议接口：

```go
 type TxAggregator interface {
     AggregateByTx(ctx context.Context, chainID uint64, txHash string) (*TxSummary, error)
     AggregateRange(ctx context.Context, chainID uint64, fromBlock, toBlock uint64) error
 }
```

### 21.11 `cursor_manager` 模块

职责：

- 统一读写 `scanner_cursors`
- 为各阶段提供断点续跑能力

建议接口：

```go
 type CursorManager interface {
     Get(ctx context.Context, scannerName string, chainID uint64, cursorType string) (uint64, error)
     Save(ctx context.Context, scannerName string, chainID uint64, cursorType string, blockNumber uint64) error
 }
```

### 21.12 `store` 模块

职责：

- 隔离数据库细节
- 给上层模块统一提供 repository 接口

建议不要让 extractor / verifier 直接操作 SQL。

建议接口按表拆分：

```go
 type TransactionStore interface {
     UpsertObservedTransactions(ctx context.Context, txs []ObservedTransaction) error
     ListObservedTransactionsByBlockRange(ctx context.Context, chainID uint64, fromBlock, toBlock uint64) ([]ObservedTransaction, error)
 }

 type InteractionStore interface {
     UpsertInteractions(ctx context.Context, items []CandidateInteraction) error
     UpdateVerificationResult(ctx context.Context, result VerifiedInteraction) error
     ListCandidateInteractions(ctx context.Context, chainID uint64, protocol string, fromBlock, toBlock uint64) ([]CandidateInteraction, error)
 }

 type LegStore interface {
     ReplaceInteractionLegs(ctx context.Context, interactionID string, legs []InteractionLeg) error
 }

 type TxSummaryStore interface {
     UpsertTxSummary(ctx context.Context, summary TxSummary) error
 }
```

### 21.13 推荐的 orchestrator

为了把上面模块串起来，建议有一个总控模块：

- `scanner_runner`

职责是：

- 按链执行各阶段
- 控制阶段顺序
- 控制 batch 范围
- 处理失败重试与 cursor 更新

建议顺序：

1. `raw_indexer`
2. `tx_fetcher`
3. `candidate_extractor`
4. `interaction_verifier`
5. `tx_aggregator`

### 21.14 如果映射到当前 `flashloan-scanner` 目录结构

如果基于当前仓库改造，建议目录方向如下：

- `synchronizer/` 保留为 `raw_indexer` 核心
- 新增 `scanner/registry/`
- 新增 `scanner/fetcher/`
- 新增 `scanner/extractor/`
- 新增 `scanner/verifier/`
- 新增 `scanner/aggregator/`
- 新增 `scanner/orchestrator/`
- `database/` 增加新表 repository

而下面这些建议逐步淡出：

- `event/contracts/`
- `relayer/`
- `worker/`
- `service/`

### 21.15 当前建议的实现顺序

如果真正开始写代码，建议顺序是：

1. 先写 `registry`
2. 再写 `store`
3. 再写 `tx_fetcher`
4. 再写 `AaveV3CandidateExtractor`
5. 再写 `AaveV3Verifier`
6. 再写 `tx_aggregator`
7. 然后扩到 Balancer / Uniswap V2

原因是：

- Aave 最适合作为第一条完整流水线模板
- 跑通一条链路后，再复制框架到其它协议，返工最少

### 21.16 下一步

完成模块接口之后，下一步最合适的是：

- 定代码目录结构
- 定第一版 migration 是否要真正落到 `flashloan-scanner/migrations`
- 再决定是否开始写实现代码

## 22. 当前代码实现进度（2026-03-24）

截至目前，`flashloan-scanner` 侧已经不只是讨论和草案，已经落了第一版 scanner 实现骨架，并且 Aave V3 路径已经具备“可串起来跑”的最小流水线。

### 22.1 已经完成的结构性改动

- 已新增 scanner migration：
  - `flashloan-scanner/migrations/00002_create_flashloan_scanner_tables.sql`
- 已新增 scanner 相关 repository：
  - `database/scanner/protocol_registry.go`
  - `database/scanner/results.go`
  - `database/scanner/cursor.go`
- 已把新表接进 `database.DB`
- 已新增 scanner 目录骨架：
  - `scanner/registry/`
  - `scanner/fetcher/`
  - `scanner/extractor/`
  - `scanner/verifier/`
  - `scanner/aggregator/`
  - `scanner/orchestrator/`
  - `scanner/cursor/`

### 22.2 Aave V3 已完成的最小流水线

当前 Aave V3 路径已经覆盖：

1. `EthereumTxFetcher`
   - 按交易哈希抓取 tx / receipt
   - 按区块范围从已有 event 表中抽取 tx hash 再补抓交易
   - 将结果写入 `observed_transactions`

2. `AaveV3CandidateExtractor`
   - 仅针对官方 Aave Pool
   - 解析 tx input
   - 识别 `flashLoan(...)`
   - 识别 `flashLoanSimple(...)`
   - 产出 candidate interaction 与 candidate legs

3. `AaveV3Verifier`
   - 基于成功交易状态、Aave `FlashLoan` event 和 leg 信息做第一版验证
   - 区分 `strict full repayment` 与 `debt opening`
   - 输出 verified interaction 与 verified legs

4. `SimpleTxAggregator`
   - 将同一 tx 下的 interaction 聚合为 `flashloan_transactions`
   - 统计 verified / strict / protocol count

5. `ProtocolRunner`
   - 已能串起：
     - `fetch range`
     - `load observed txs`
     - `extract candidates`
     - `persist interactions`
     - `persist legs`
     - `verify interactions`
     - `re-write verified legs`
     - `aggregate tx summary`

### 22.3 当前 runner 的运行方式

`ProtocolRunner` 当前支持：

- `RunOnce(ctx, chainID, fromBlock, toBlock)`
- `RunLoop(ctx, chainID)`

其中 `RunLoop` 这一版已经补了：

- 最新区块感知
- scanner cursor 管理
- 固定 batch size 分段推进
- 追平后 sleep 再继续轮询

但它当前仍属于“最小可用版”：

- 还没有 confirmation depth
- 还没有 trace 验证
- 还没有失败重试分层策略
- 还没有多协议统一调度器

### 22.4 当前验证强度

目前 Aave V3 的 verifier 是：

- **event-backed verification**

不是：

- **trace-backed strict verification**

也就是说，这一版已经能做：

- candidate 抽取
- 基础 verified 判定
- strict 与 debt-opening 的一阶区分

但还没有做到最强口径下的：

- callback trace 证据
- reserve / aToken repayment trace 证据

### 22.5 当前测试状态

截至 2026-03-24，本地已通过：

```bash
go test ./scanner/... ./database/scanner ./database/event
```

通过的包包括：

- `scanner/aggregator`
- `scanner/orchestrator`
- `scanner/verifier`

其余 scanner 子目录当前主要是骨架和实现代码，暂时还没有更多单测。

### 22.6 下一步最合理的实现顺序

从当前状态往前推进，最合理的顺序是：

1. 给 Aave runner 增加 seed / bootstrap，先把官方 Pool 地址灌进 `protocol_addresses`
2. 再补一个真正的 scanner 装配入口，把 `DB + RPC + registry + runner` 组起来
3. 然后扩 Balancer V2
4. 最后再做 Uniswap V2

原因是：

- Aave 已经有完整骨架
- 现在最缺的是“如何实际跑起来”
- Balancer 能复用大部分框架
- Uniswap V2 对 trace 和 pair 管理要求最高，应该最后做

### 22.7 新增的 seed / bootstrap 与装配能力

在上一轮最小流水线基础上，这一轮又补了两类运行时能力：

1. 地址 seed / bootstrap
   - 新增 `scanner/bootstrap/seed.go`
   - 已支持将官方 Aave V3 Pool 地址批量写入 `protocol_addresses`
   - 当前入口：
     - `SeedOfficialAaveV3Pools(...)`
     - `SeedOfficialProtocolAddresses(...)`

2. Aave V3 runner 装配
   - 新增 `scanner/orchestrator/aave_v3_builder.go`
   - 已支持基于：
     - `database.DB`
     - `*rpc.Client`
     - `chainID`
     - `AaveV3RunnerConfig`
   - 直接组装出一条可运行的 Aave V3 scanner pipeline

这层 builder 当前会自动装配：

- DB-backed registry
- GORM store
- Ethereum tx fetcher
- Aave V3 candidate extractor
- Aave V3 verifier
- Simple tx aggregator
- GORM cursor manager
- `ProtocolRunner`

也就是说，scanner 侧现在已经不是“只有模块，没有装配”，而是已经具备了一条可实例化的 Aave 路径。

### 22.8 当前仍未完成的运行时缺口

虽然现在已经能 seed 地址并装配 runner，但还缺下面几项，才算真正能在服务里长期跑：

- 主程序或独立命令入口
- scanner 专用配置结构
- confirmation depth 接入
- trace RPC 接入
- Aave 官方地址的默认种子来源
- Balancer / Uniswap V2 的对应 builder

### 22.9 代码层下一步建议

如果继续按最短路径推进，下一步最值的是：

1. 新增一个独立 scanner 启动入口
   - 不要先硬接 `FlashloanScannerApp.Start()`
   - 先做独立 command 或最小 service

2. 给配置加 scanner 段
   - `enabled`
   - `protocol`
   - `start_block`
   - `batch_size`
   - `loop_interval_seconds`
   - `aave_pools`

3. 用配置驱动 seed + builder + run loop

这样你们就能真正开始“扫链”，而不是停留在模块实现阶段。

### 22.10 独立 scanner 启动入口（已完成第一版）

这一轮已经把 flash-loan scanner 从“模块集合”推进到了“可单独启动”的程度。

#### 已完成内容

1. 配置层
   - `config/config.go` 已新增 `scanner` 配置段
   - 当前已支持字段：
     - `enabled`
     - `protocol`
     - `chain_id`
     - `run_mode`
     - `start_block`
     - `end_block`
     - `batch_size`
     - `loop_interval_seconds`
     - `aave.pools`
   - 另补了 `RPCByChainID(...)` 帮助方法

2. CLI 入口
   - `cmd/flashloan-scanner/cli.go` 已新增子命令：
     - `flashloan-scan`
   - 这意味着当前二进制已经能独立启动 scanner，而不需要复用桥接 indexer 流程

3. Scanner lifecycle service
   - 已新增 `scanner/app/service.go`
   - 当前 service 已能：
     - 读取 scanner 配置
     - 初始化 DB
     - 连接 RPC
     - seed 官方 Aave Pool 地址
     - build Aave V3 runner
     - 根据 `run_mode` 执行：
       - `once`
       - `loop`

4. 示例配置
   - 已在 `flashloan-scanner.example.yaml` 追加 scanner 配置段示例

#### 当前可用的执行方式

理论上现在已经可以使用：

```bash
flashloan-scanner flashloan-scan --config ./flashloan-scanner.yaml
```

但前提是：

- 数据库已迁移
- 对应链的 `contract_events_<chain_id>` / `block_headers_<chain_id>` 已经有数据
- `scanner.aave.pools` 已配置官方 Pool 地址
- `scanner.protocol` 当前必须是 `aave_v3`

#### 当前限制

当前这个独立入口仍然只支持：

- Aave V3

还不支持：

- Balancer V2
- Uniswap V2
- trace-backed strict verification
- confirmation depth 参数化
- 多协议统一调度

### 22.11 Balancer V2 已接入第一版实现

这一轮已经把第二个协议接进 scanner 主链路，当前支持的第二个协议是：

- `balancer_v2`

#### 已完成模块

1. ABI
   - 新增 `scanner/balancer/abi.go`
   - 当前包含最小 Vault ABI：
     - `flashLoan(...)`
     - `FlashLoan` event

2. Candidate extractor
   - 新增 `scanner/extractor/balancer_v2.go`
   - 当前逻辑：
     - 仅识别官方 Vault
     - 解析 `flashLoan(recipient, tokens, amounts, userData)`
     - 生成 interaction
     - 生成多 token legs

3. Verifier
   - 新增 `scanner/verifier/balancer_v2.go`
   - 当前是 event-backed verifier：
     - 交易必须成功
     - 必须看到 Vault `FlashLoan` event
     - 必须逐个 leg 匹配 recipient / token / amount
     - 匹配成功后记为 `verified + strict`

4. Builder
   - 新增 `scanner/orchestrator/balancer_v2_builder.go`
   - 已可按 Aave 相同方式装配出 Balancer V2 runner

5. Seed / service switch
   - `scanner/bootstrap/seed.go` 已新增：
     - `SeedOfficialBalancerV2Vaults(...)`
   - `scanner/app/service.go` 已支持：
     - `scanner.protocol = balancer_v2`
     - `scanner.balancer.vaults`

6. 示例配置
   - `flashloan-scanner.example.yaml` 已增加：
     - `scanner.balancer.vaults`

#### 当前 Balancer V2 的验证强度

和当前 Aave 一样，Balancer V2 目前仍然是：

- **event-backed verification**

这意味着现在能做：

- 候选提取
- 基础 verified / strict 标记
- leg 级别 fee 记录

但还没有做到：

- trace-backed callback 证据
- Vault 回调路径级验证

#### 当前已通过测试

截至 2026-03-24，本地已通过：

```bash
go test ./cmd/flashloan-scanner ./config ./scanner/... ./database/scanner ./database/event
```

其中新增通过的相关包包括：

- `scanner/bootstrap`
- `scanner/orchestrator`
- `scanner/verifier`
- `scanner/app`

#### 当前协议支持状态

截至目前，`flashloan-scan` 命令已支持：

- `aave_v3`
- `balancer_v2`

仍未接入：

- `uniswap_v2`

### 22.12 Uniswap V2 已接入第一版实现

这一轮已经把第三个协议接进 scanner 主链路，当前支持的第三个协议是：

- `uniswap_v2`

#### 已完成模块

1. ABI
   - 新增 `scanner/uniswapv2/abi.go`
   - 当前包含最小 Pair ABI：
     - `swap(...)`
     - `Swap` event

2. Candidate extractor
   - 新增 `scanner/extractor/uniswap_v2.go`
   - 当前逻辑：
     - 仅识别已登记为官方的 pair
     - 解析 `swap(amount0Out, amount1Out, to, data)`
     - 仅当 `data != empty` 时生成 flash-swap candidate
     - 基于已 seed 的 pair 元数据补齐：
       - `factory_address`
       - `pair_address`
       - `token0`
       - `token1`

3. Verifier
   - 新增 `scanner/verifier/uniswap_v2.go`
   - 当前是 event-backed verifier：
     - 交易必须成功
     - candidate 必须来自 `data != empty` 的 swap
     - 必须看到 pair 的 `Swap` event
     - event 中的 `amount0Out / amount1Out` 必须与 legs 匹配
   - 当前会把：
     - `callback_seen = true`
     - `settlement_mode = invariant_restored`
   - 这些都属于**基于成功 pair.swap + non-empty data + matching Swap event 的推断**，不是 trace 级证据

4. Builder
   - 新增 `scanner/orchestrator/uniswap_v2_builder.go`
   - 已可按 Aave / Balancer 相同方式装配出 Uniswap V2 runner

5. Seed / service switch
   - `scanner/bootstrap/seed.go` 已新增：
     - `SeedOfficialUniswapV2Factories(...)`
     - `SeedOfficialUniswapV2Pairs(...)`
   - `scanner/app/service.go` 已支持：
     - `scanner.protocol = uniswap_v2`
     - `scanner.uniswap_v2.factories`
     - `scanner.uniswap_v2.pairs`

6. 示例配置
   - `flashloan-scanner.example.yaml` 已增加：
     - `scanner.uniswap_v2.factories`
     - `scanner.uniswap_v2.pairs`

#### 当前 Uniswap V2 的实现边界

这一版 Uniswap V2 仍然是“最小可运行版”，边界非常明确：

- 只支持**手工 seed 的官方 pair 元数据**
- 不支持自动 pair discovery
- 不支持 V2 forks
- 不支持 trace-backed callback 验证
- 不支持 pair 储备级 invariant 复算

#### 当前协议支持状态

截至 2026-03-24，`flashloan-scan` 已支持：

- `aave_v3`
- `balancer_v2`
- `uniswap_v2`

这意味着三协议的第一版 scanner 骨架已经全部接通。

### 22.13 本地 smoke 路径（已完成第一版）

这一轮已经把“讨论中的 smoke 路径”落成了仓库内可执行脚本。

#### 已完成内容

1. PowerShell smoke script
   - 新增 `flashloan-scanner/scripts/flashloan-smoke.ps1`
   - 当前串联步骤：
     - `migrate`
     - `flashloan-scan`
   - 支持参数：
     - `-Config`
     - `-MigrationsDir`
     - `-SkipMigrate`

2. README 补充
   - `flashloan-scanner/README.md` 已新增 flashloan scanner smoke path 说明
   - 已写明当前依赖条件：
     - scanner 配置必须开启
     - `run_mode` 建议为 `once`
     - 依赖已有原始链上数据表

#### 当前推荐执行方式

```powershell
pwsh -NoProfile -File .\scripts\flashloan-smoke.ps1 -Config .\flashloan-scanner.local.yaml
```

#### 当前 smoke 路径的现实边界

这个脚本现在只是把：

- 数据库 migration
- scanner 一次性运行

串成一条路径。

它不会帮你自动完成：

- 本地 Postgres 启动
- 原始 block / event 数据灌库
- 官方地址自动发现
- Uniswap pair 自动发现

所以它当前是：

- **scanner command-chain smoke path**

还不是：

- **end-to-end chain replay bootstrap**

### 22.14 本地 fixture 导入路径（已完成第一版）

为了让 smoke 不依赖真实 RPC 上的历史 tx 补抓，这一轮又补了一条本地 fixture 路径。

#### 已完成内容

1. Fixture builder / loader
   - 新增 `scanner/fixture/aave_v3.go`
   - 当前已支持构造并写入一条本地 Aave V3 `flashLoanSimple` 样本，涉及：
     - `block_headers_<chain_id>`
     - `contract_events_<chain_id>`
     - `observed_transactions`

2. Fixture CLI
   - `cmd/flashloan-scanner/cli.go` 已新增子命令：
     - `flashloan-fixture`
   - 当前只支持：
     - `aave_v3`

3. Skip-fetch 运行模式
   - `config.Scanner` 已新增：
     - `skip_tx_fetch`
   - `ProtocolRunner` 已支持跳过 `FetchRange(...)`
   - 这使得 scanner 可以直接消费 fixture 写入的 `observed_transactions`

4. Smoke script 集成
   - `scripts/flashloan-smoke.ps1` 已新增：
     - `-LoadFixture`
   - 如果使用 fixture，本地推荐路径是：

```powershell
pwsh -NoProfile -File .\scripts\flashloan-smoke.ps1 -Config .\flashloan-scanner.local.yaml -LoadFixture
```

#### 当前使用前提

若要使用内置 fixture 路径，当前配置必须满足：

- `scanner.enabled: true`
- `scanner.protocol: aave_v3`
- `scanner.run_mode: once`
- `scanner.skip_tx_fetch: true`
- `scanner.aave.pools` 至少配置一个 pool 地址

#### 当前实现边界

这一版 fixture 路径仍然只是：

- **Aave V3 单样本本地 smoke fixture**

还不是：

- 多协议 fixture 集
- 自动生成 trace fixture
- 真实链上样本 replay

### 22.15 Aave trace-backed strict verification（第一版已接入）

这一轮已经把 Aave 的第一版 trace 能力接进 scanner。

#### 已完成内容

1. trace provider 抽象
   - 新增 `scanner/trace/trace.go`
   - 新增 `scanner/trace/geth.go`
   - 当前实现基于：
     - `debug_traceTransaction`
     - `callTracer`

2. Aave trace-aware verifier
   - 新增 `scanner/verifier/aave_v3_trace.go`
   - 当前逻辑是：
     - 先跑原有 `event-backed` Aave verifier
     - 再补 trace 级证据

3. 当前 trace 证据口径
   - callback 证据：
     - trace 中存在 `Pool -> receiver` 的成功调用
   - repayment-path 证据：
     - trace 中存在与借贷资产地址相关的 `approve / transfer / transferFrom` 调用
     - 或存在 provider 与资产地址相关的成功调用
   - debt opening：
     - 仍然优先沿用 event/input 层对 `interestRateMode` 的判断

4. builder / config 已接入
   - `scanner/orchestrator/aave_v3_builder.go` 已支持 `TraceEnabled`
   - `config.Scanner` 已新增 `trace_enabled`
   - `scanner/app/service.go` 已把这个开关接进 Aave builder

#### 当前实现的实际含义

这不是“完整资金流审计”，而是把 Aave strict verification 从：

- 纯日志推断

提升到：

- 日志验证 + 调用路径证据

也就是说，这一版更关注：

- callback 是否真的发生
- repayment path 是否有 trace 证据

而不是精确重建全部 reserve / aToken 资金流。

#### 当前边界

这一版 trace verifier 仍然存在明确边界：

- 只接入 Aave V3
- 默认 trace provider 假设节点支持 `debug_traceTransaction(callTracer)`
- repayment-path 判断仍然是 heuristic，不是完整 accounting
- 还没有对不同客户端 trace 格式做兼容层

#### 当前建议用法

如果后面要启用它，配置里应设置：

- `scanner.trace_enabled: true`

但在真正启用前，最好先确认目标 RPC 是否开放 `debug_traceTransaction`。

#### 当前验证状态

这一轮只做了代码接入和格式化，没有继续运行测试。

### 22.16 Aave trace verifier 单测与使用约束（已补充）

这一轮又把 Aave trace 层往前推进了一步：

#### 已新增单测

新增文件：

- `scanner/verifier/aave_v3_trace_test.go`

当前覆盖了 3 个核心场景：

1. **trace strict success**
   - event-backed verifier 先通过
   - trace 中存在 `Pool -> receiver` callback
   - trace 中存在 repayment-path 证据
   - 最终保持 `verified = true, strict = true`

2. **missing callback**
   - event-backed verifier 先通过
   - 但 trace 中不存在 `Pool -> receiver` callback
   - 最终降为：
     - `verified = false`
     - `strict = false`
     - `exclusion_reason = missing_trace_callback`

3. **callback but no repayment-path**
   - callback 存在
   - 但 repayment-path 证据不足
   - 最终保持：
     - `verified = true`
   - 但降为：
     - `strict = false`

#### README 约束已补充

README 现在明确写了：

- `scanner.trace_enabled: true` 当前只对 `aave_v3` 有意义
- 启用 trace 需要目标 RPC 支持：
  - `debug_traceTransaction`
  - `callTracer`
- 本地 fixture smoke 不会注入 trace 数据，因此 fixture 模式下应保持：
  - `scanner.trace_enabled: false`

#### 当前状态总结

到这里，Aave 的 strict verification 已分成两层：

- event-backed baseline
- trace-backed strengthened strict verification

这意味着后面如果你们需要在报告里写“严格口径”和“弱口径”的差异，代码上已经有对应落点可用了。

### 22.17 Balancer trace-backed strict verification（第一版已接入）

这一轮已经把 Balancer 的第一版 trace 能力也接进 scanner。

#### 已完成内容

1. trace-aware verifier
   - 新增 `scanner/verifier/balancer_v2_trace.go`
   - 逻辑模式与 Aave 保持一致：
     - 先跑 event-backed Balancer verifier
     - 再补 trace 级证据

2. 当前 trace 证据口径
   - callback 证据：
     - trace 中存在 `Vault -> recipient` 的成功调用
   - repayment-path 证据：
     - trace 中存在与借贷 token 地址相关的 `approve / transfer / transferFrom` 调用痕迹

3. builder / service 已接入
   - `scanner/orchestrator/balancer_v2_builder.go` 已支持 `TraceEnabled`
   - `scanner/app/service.go` 已将 `scanner.trace_enabled` 传入 Balancer builder

4. 已补单测
   - 新增 `scanner/verifier/balancer_v2_trace_test.go`
   - 当前覆盖：
     - trace strict success
     - missing callback
     - callback but no repayment-path

#### 当前状态

到这里，`scanner.trace_enabled` 在代码上已经对两类协议生效：

- `aave_v3`
- `balancer_v2`

仍未接入 trace-aware verification 的协议：

- `uniswap_v2`

### 22.18 Uniswap V2 trace-backed strict verification（第一版已接入）

这一轮已经把 Uniswap V2 的第一版 trace 能力也接进 scanner。

#### 已完成内容

1. trace-aware verifier
   - 新增 `scanner/verifier/uniswap_v2_trace.go`
   - 逻辑模式与 Aave / Balancer 一致：
     - 先跑 event-backed Uniswap V2 verifier
     - 再补 trace 级证据

2. 当前 trace 证据口径
   - callback 证据：
     - trace 中存在 `pair -> receiver` 的成功调用
   - invariant-path 证据：
     - trace 中存在与 `token0 / token1` 资产地址相关的 `approve / transfer / transferFrom` 调用痕迹

3. builder / service 已接入
   - `scanner/orchestrator/uniswap_v2_builder.go` 已支持 `TraceEnabled`
   - `scanner/app/service.go` 已将 `scanner.trace_enabled` 传入 Uniswap V2 builder

4. 已补单测
   - 新增 `scanner/verifier/uniswap_v2_trace_test.go`
   - 当前覆盖：
     - trace strict success
     - missing callback
     - callback but no invariant-path

#### 当前状态

到这里，`scanner.trace_enabled` 在代码上已经对三协议都生效：

- `aave_v3`
- `balancer_v2`
- `uniswap_v2`

这意味着三协议现在都具备：

- event-backed baseline
- trace-backed strengthened strict verification

### 22.19 结果消费层：flashloan-report（已接入）

这一轮已经把结果消费层补进 CLI，不再只有“写库”，也能“读库”。

#### 已完成内容

1. 数据库查询能力
   - `database/scanner/results.go` 已新增：
     - `ListFlashloanTransactions(chainID, onlyStrict, limit)`

2. report service
   - 新增 `scanner/report/report.go`
   - 当前可按：
     - `chain_id + limit`
     - `chain_id + tx_hash`
   - 读取：
     - `flashloan_transactions`
     - `protocol_interactions`
     - `interaction_asset_legs`

3. CLI 命令
   - `cmd/flashloan-scanner/cli.go` 已新增：
     - `flashloan-report`
   - 当前支持参数：
     - `--config`
     - `--chain-id`
     - `--tx-hash`
     - `--limit`
     - `--strict-only`

4. 已补基础测试
   - 新增 `scanner/report/report_test.go`
   - 当前只覆盖文本渲染层，不涉及真实数据库

#### 当前作用

到这里，scanner 代码已经形成：

- 结果写入
- 结果读取
- 命令行查看

这意味着你们后面不需要每次手写 SQL 才能看结果了，至少已经有一个最小结果查看入口。

### 22.20 结果消费层：SQL 查询模板（已补充）

为了让 scanner 结果除了 CLI 之外还能直接做数据库分析，这一轮补了一份可复用的 PostgreSQL 查询模板文件：

- `flashloan-scanner/sql/flashloan_report_queries.sql`

#### 当前已覆盖的查询场景

1. 最近的 flash-loan 交易
2. 最近的 strict-only 交易
3. 单笔交易 drilldown
4. 单笔交易 + asset legs drilldown
5. 协议级 interaction 计数
6. candidate / verified / strict 总览
7. exclusion reason 排序
8. 已 verified 但未 strict 的 interaction
9. 按协议和资产统计 borrowed volume
10. 带 trace 相关说明的 strict interactions

#### 当前作用

这份 SQL 模板的作用不是替代 `flashloan-report`，而是补齐两类场景：

- 想快速在 psql / DataGrip / DBeaver 里直接查数
- 想把方法部分对应到明确的数据库查询语句

这样到这里为止，结果消费层已经有两条路径：

- CLI：`flashloan-report`
- SQL：`flashloan_report_queries.sql`

## 23. 当前项目状态总结（课程项目口径）

到目前为止，这个 flash-loan scanner 已经完成了第一版可交付实现，可以作为研究/课程项目的主体成果使用。

### 23.1 当前已完成的部分

1. 协议范围
   - 已支持三类协议：
     - `aave_v3`
     - `balancer_v2`
     - `uniswap_v2`

2. 识别流程
   - 已完成三协议统一的 `candidate -> verified -> strict` 识别流程
   - 已落地 `interaction / asset leg / transaction` 三层数据粒度

3. 验证层
   - 已完成 `event-backed` 第一层验证
   - 已完成 `trace-backed strengthened strict verification` 第二层验证
   - `scanner.trace_enabled` 目前已对三协议生效

4. 数据层
   - 已完成 scanner 结果表、支撑表、migration 和 repository
   - 已完成结果写入、读取、聚合与导出

5. 运行层
   - 已新增独立命令：
     - `flashloan-scan`
     - `flashloan-fixture`
     - `flashloan-report`
   - 已完成本地 fixture 路径和 smoke 脚本

6. 结果消费层
   - 已支持 CLI 文本查看
   - 已支持 `json / csv` 导出
   - 已补充 PostgreSQL 查询模板：
     - `flashloan-scanner/sql/flashloan_report_queries.sql`

### 23.2 当前可以怎样表述完成度

如果按研究/课程项目口径，这一版已经可以表述为：

- 已完成 flash-loan scanner 的第一版系统设计与实现
- 已完成三协议模式化识别
- 已完成候选识别、验证、严格验证和结果持久化
- 已完成基本结果查看与导出能力

换句话说，这一版已经不是只有“方法设计”，而是已经形成了可运行的原型系统（prototype implementation）。

### 23.3 还未完成或仍需诚实说明的部分

虽然第一版已经闭环，但还不能把它表述成“最终完善系统”或“生产级实现”。当前仍有这些边界：

1. 真实链上端到端验证还未完全跑通
   - 之前本地 smoke 运行卡在 PostgreSQL 认证，不是代码逻辑错误，但意味着尚未完成完整实跑验证记录

2. trace-backed strict verification 仍是第一版增强验证
   - 当前 trace 验证已经明显强于纯 event 推断
   - 但还没有做到完整资金流审计级别的 repayment / accounting 校验

3. Uniswap V2 仍偏配置驱动
   - 当前实现更偏 seed / config 驱动
   - 还没有做成完整自动 pair discovery 和持续维护闭环

4. 测试覆盖仍以模块测试为主
   - 当前已有 extractor / verifier / report 等针对性测试
   - 但还不是完整的集成测试矩阵

5. 多协议统一调度仍是第一版
   - 当前已能按协议独立运行
   - 但还没有进一步扩展成更成熟的统一调度系统

### 23.4 课程项目中建议采用的表述

如果写进课程项目说明，建议使用下面这类表述：

- 本项目已实现一个面向 Aave V3、Balancer V2 和 Uniswap V2 的 flash-loan transaction scanner 原型系统。
- 系统能够对链上交易执行候选识别、协议特化验证和严格验证，并将结果落库为 interaction、asset leg 和 transaction 三层结构化数据。
- 在验证层面，系统同时实现了基于事件的第一层验证，以及基于 transaction trace 的增强严格验证。

不要直接写成：

- 已实现完全准确的生产级检测系统
- 已完成所有真实链上数据的全面验证

更稳的说法是：

- 已完成第一版 prototype implementation
- 已实现研究所需的核心识别与验证能力
- 后续仍可继续增强 trace/accounting 校验与真实数据实验规模

### 23.5 当前阶段结论

截至目前，这个项目最准确的状态是：

- 对课程/研究项目而言，主体代码已经完成
- 对工程化和生产化而言，还存在后续增强空间

因此，后续工作重点不必再放在“从零补系统”，而应转向：

- 真实数据实验
- 结果分析
- 报告撰写
- 对当前限制进行规范说明

### 23.6 本地最小链路已真实跑通

这一轮已经完成了一次本地真实执行验证，不再只是“代码和测试存在”，而是实际把最小链路跑通了。

#### 本次实际执行的链路

1. 本地 PostgreSQL 数据库已创建并可连接
   - 数据库名：`flashloan_scanner`
   - 本地配置已切到 `postgres / flashloan_scanner`

2. 已成功执行 migration
   - scanner 相关结果表与支撑表已在本地数据库创建成功

3. 已成功执行本地 fixture + scanner
   - 执行命令：
     - `pwsh -NoProfile -File .\scripts\flashloan-smoke.ps1 -Config .\flashloan-scanner.local.yaml -MigrationsDir .\migrations -LoadFixture`
   - 该链路完成了：
     - `migrate`
     - `flashloan-fixture`
     - `flashloan-scan`

4. 已成功执行结果查看
   - 执行命令：
     - `go run .\cmd\flashloan-scanner flashloan-report --config .\flashloan-scanner.local.yaml --chain-id 11155111 --limit 5`

#### 本次实际得到的结果

当前本地数据库中已经能看到一条由 Aave fixture 生成并被 scanner 识别出的结果，最终状态为：

- `candidate = true`
- `verified = true`
- `strict = true`
- `protocol = aave_v3`

也就是说，下面这条最小闭环已经被真实执行验证：

- Aave fixture 写库
- scanner 候选识别
- verifier 完成验证
- interaction / legs / tx summary 落库
- report 成功读出结果

#### 这一步的意义

这意味着当前项目已经不仅仅是“设计完成”或“模块代码存在”，而是已经完成了一次可复现的本地 prototype run。

对课程项目而言，这一点很重要，因为它可以支持你们在报告中更稳地表述：

- 系统已经实现并完成本地端到端原型验证
- 至少在 Aave fixture 场景下，完整识别链路可以成功执行

#### 仍需注意的边界

这里跑通的是：

- 本地数据库
- 本地 Aave fixture
- 本地 scanner 原型链路

还不是：

- 大规模真实主网/测试网数据实验
- 完整真实链上数据集评估

因此更准确的说法是：

- 本地最小原型链路已真实跑通
- 真实链上实验仍属于后续工作



### 23.7 Balancer V2 本地最小链路已真实跑通

在 Aave 本地链路验证完成之后，这一轮继续补齐了 Balancer V2 的本地 fixture，并完成了一次独立的本地端到端验证。

#### 本次新增内容

1. 已新增 Balancer V2 fixture 构造与写库逻辑
   - 新增文件：
     - `flashloan-scanner/scanner/fixture/balancer_v2.go`
     - `flashloan-scanner/scanner/fixture/balancer_v2_test.go`

2. 已将 `flashloan-fixture` 命令扩展到支持 `balancer_v2`
   - 当 `scanner.protocol = balancer_v2` 时，可自动写入一笔本地 Vault `flashLoan(...)` 样本

3. 已新增独立本地配置
   - `flashloan-scanner/flashloan-scanner.balancer.local.yaml`
   - 该配置使用：
     - `scanner.protocol = balancer_v2`
     - `scanner.skip_tx_fetch = true`
     - `scanner.trace_enabled = false`

#### 本次实际执行的命令

1. 本地 smoke 执行：
   - `pwsh -NoProfile -File .\scripts\flashloan-smoke.ps1 -Config .\flashloan-scanner.balancer.local.yaml -MigrationsDir .\migrations -LoadFixture`

2. 结果查看：
   - `go run .\cmd\flashloan-scanner flashloan-report --config .\flashloan-scanner.balancer.local.yaml --chain-id 11155111 --limit 5`

#### 本次实际得到的结果

当前本地数据库中已经能看到一条由 Balancer V2 fixture 生成并被 scanner 识别出的结果，最终状态为：

- `candidate = true`
- `verified = true`
- `strict = true`
- `protocol = balancer_v2`

这意味着下面这条闭环已经被真实执行验证：

- Balancer V2 fixture 写库
- scanner 候选识别
- verifier 完成验证
- interaction / legs / tx summary 落库
- report 成功读出结果

#### 当前三协议验证状态

截至目前：

- `Aave V3` 已完成本地最小链路真实运行验证
- `Balancer V2` 已完成本地最小链路真实运行验证
- `Uniswap V2` 已完成代码实现，但尚未完成同等程度的本地 fixture 运行验证

因此，当前项目已经不再只是“单协议 prototype 可跑”，而是已经完成了两条协议的本地端到端原型验证。

### 23.8 Uniswap V2 本地最小链路已真实跑通

在 Aave 与 Balancer V2 的本地链路验证完成之后，这一轮继续补齐了 Uniswap V2 的本地 fixture，并完成了一次独立的本地端到端验证。

#### 本次新增内容

1. 已新增 Uniswap V2 fixture 构造与写库逻辑
   - 新增文件：
     - `flashloan-scanner/scanner/fixture/uniswap_v2.go`
     - `flashloan-scanner/scanner/fixture/uniswap_v2_test.go`

2. 已将 `flashloan-fixture` 命令扩展到支持 `uniswap_v2`
   - 当 `scanner.protocol = uniswap_v2` 时，可自动写入一笔本地 official pair `swap(data != empty)` 样本

3. 已新增独立本地配置
   - `flashloan-scanner/flashloan-scanner.uniswap.local.yaml`
   - 该配置使用：
     - `scanner.protocol = uniswap_v2`
     - `scanner.skip_tx_fetch = true`
     - `scanner.trace_enabled = false`
     - `scanner.uniswap_v2.pairs` 预置一条本地 pair 元数据

#### 本次实际执行的命令

1. 本地 smoke 执行：
   - `pwsh -NoProfile -File .\scripts\flashloan-smoke.ps1 -Config .\flashloan-scanner.uniswap.local.yaml -MigrationsDir .\migrations -LoadFixture`

2. 结果查看：
   - `go run .\cmd\flashloan-scanner flashloan-report --config .\flashloan-scanner.uniswap.local.yaml --chain-id 11155111 --limit 10`

#### 本次实际得到的结果

当前本地数据库中已经能看到一条由 Uniswap V2 fixture 生成并被 scanner 识别出的结果，最终状态为：

- `candidate = true`
- `verified = true`
- `strict = true`
- `protocol = uniswap_v2`

这意味着下面这条闭环已经被真实执行验证：

- Uniswap V2 fixture 写库
- scanner 候选识别
- verifier 完成验证
- interaction / legs / tx summary 落库
- report 成功读出结果

#### 当前三协议验证状态

截至目前，三条协议都已经完成了本地最小链路真实运行验证：

- `Aave V3`
- `Balancer V2`
- `Uniswap V2`

因此，对课程项目而言，可以更稳地表述为：

- 三协议识别原型均已实现
- 三协议均已完成至少一次本地端到端原型验证
- 当前仍未进行大规模真实链上实验，真实数据评估属于后续工作

### 23.9 已定位一段真实主网区间，包含三协议样本

在完成三协议本地 fixture 验证之后，这一轮继续尝试从真实 Ethereum mainnet 数据中定位一段同时包含 Aave V3、Balancer V2 与 Uniswap V2 样本的区块区间。

#### 推荐区间

- 推荐扫描区间：`22485844` 到 `22486844`
- 更紧的三笔样本覆盖区间：`22485972` 到 `22486344`

#### 当前已确认的三笔真实样本

1. Aave V3
   - tx: `0xbe7587949e33104beb827d06ebc83c71c8fa560128152241a52f62d4f7f6daa0`
   - block: `22485972`
   - provider: `0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2`

2. Balancer V2
   - tx: `0xcfb9ce1213d710127eadd563ae61d58d8a35a789243ae8fee2712c05cd1110d3`
   - block: `22486336`
   - provider: `0xBA12222222228d8Ba445958a75a0704d566BF2C8`

3. Uniswap V2
   - tx: `0xbfe5657b9758b02d679eb58d6919b181cb83c4592ded9f31d2476d973b1a6165`
   - block: `22486344`
   - pair: `0xCe407CD7b95B39d3B4d53065E711e713dd5C5999`
   - 已额外确认：
     - 顶层调用目标是该 pair
     - `swap(...)` 的 `data` 非空
     - 交易成功

#### 这一节的意义

这说明当前项目不仅有本地构造样本，也已经能定位到一段真实主网区间，使后续真实数据实验可以在一个较小窗口内开展，而不必从全链盲扫开始。

#### 当前对应配置草案

已新增配置文件：

- `flashloan-scanner/flashloan-scanner.mainnet.real-window.yaml`

该配置已预填：

- `chain_id = 1`
- `start_block = 22485844`
- `end_block = 22486844`
- Aave V3 官方 Pool
- Balancer V2 官方 Vault
- 一条已确认的 Uniswap V2 real flash-swap pair sample

#### 仍需注意的边界

这一步完成的是：

- 真实主网区间定位
- 真实样本交易定位
- Uniswap V2 样本的顶层 `swap(data != empty)` 核验

还没有完成的是：

- 用当前 scanner 对这段真实主网区间做一次完整跑数
- 对真实结果做人工抽样复核
- 形成正式实验统计表

### 24. 报告可直接使用的系统实现总结

以下文字可直接作为课程项目报告中“系统实现”部分的基础稿，再按篇幅需要做删改。

#### 24.1 系统实现概述

本项目实现了一个面向 Ethereum 链上闪电贷交易识别的原型系统 `flashloan-scanner`。系统围绕三类典型 provider 设计协议特化识别器，分别覆盖 `Aave V3`、`Balancer V2` 与 `Uniswap V2`。整体流程采用两阶段识别框架：首先基于协议入口、官方地址与关键日志信号识别候选交互；随后结合协议事件与可选的交易 trace 信息，对候选样本进行验证并生成严格口径下的闪电贷标签。

在实现层面，系统主要由以下模块组成：

- `registry`：维护官方协议地址与 Uniswap V2 pair 元数据
- `fetcher`：负责按区块区间补抓交易与 receipt
- `extractor`：执行协议特化的 candidate 识别
- `verifier`：执行 `verified` 与 `strict` 判定
- `aggregator`：将 interaction 级识别结果聚合到 transaction 级
- `report`：提供文本、JSON 与 CSV 三种结果查看方式

为了适配课程项目的研究目标，系统内部同时保存三层粒度的数据：

- `interaction`：一次 provider 发起并完成的 flash-loan / flash-swap 过程
- `asset leg`：一次 interaction 中某个 token 的借入与结算情况
- `transaction`：一笔交易是否包含至少一个 verified interaction

这一设计使系统既能支持协议级模式识别，也能支持后续资产分布、费用和验证状态等统计分析。

#### 24.2 三协议识别器实现

`Aave V3` 识别器优先依据官方 Pool 的 `flashLoan(...)` 与 `flashLoanSimple(...)` 入口函数生成候选样本，并在验证阶段匹配 `FlashLoan` 事件、逐 leg 对齐 `asset / amount / receiver`，同时利用 `interestRateMode` 区分严格闪电贷与 debt-opening 场景。对于启用 trace 的运行模式，系统还会进一步检查 `Pool -> receiver` 的 callback 调用以及偿还路径相关调用痕迹，以提升严格验证的可信度。

`Balancer V2` 识别器在第一版实现中原本只接受顶层直接调用官方 Vault 的交易。为了适配真实主网中经由中间合约触发 Vault flash loan 的情况，系统进一步扩展为两条候选路径：一是顶层 `Vault.flashLoan(...)` 入口，二是同一笔交易内出现官方 Vault `FlashLoan` 事件时的事件派生候选。这样可以覆盖真实链上更常见的复杂调用结构。

`Uniswap V2` 识别器以官方 pair 的 `swap(...)` 调用为入口，仅当 `data` 非空时才生成 flash swap 候选。在验证阶段，系统结合 `Swap` 事件、pair 元数据以及可选 trace 信息，确认该交互满足 flash swap 的结算语义，而不会将普通 router swap 误判为 flash swap。

#### 24.3 工程实现特点

本项目不是简单的离线 SQL 统计，而是实现了一个可运行的原型扫描系统。系统支持：

- 独立扫描命令 `flashloan-scan`
- 本地样本写入命令 `flashloan-fixture`
- 结果查看命令 `flashloan-report`
- PostgreSQL 表结构迁移
- 本地 smoke 脚本与最小端到端运行路径

在课程项目范围内，这种实现方式的意义在于：识别规则不仅停留在方法设计层面，而是已经以代码形式落地，并能对输入区块区间输出结构化识别结果。

### 25. 报告可直接使用的实验验证与局限总结

以下文字可直接作为课程项目报告中“实验验证”与“局限性讨论”部分的基础稿。

#### 25.1 原型验证设置

在完成三协议本地 fixture 验证之后，本项目进一步选取了一段真实 Ethereum mainnet 区块区间，用于验证系统在真实链上样本上的可用性。最终选定的扫描窗口为 `22485844` 到 `22486844`，其中包含三笔已人工确认的真实样本：

- Aave V3：`0xbe7587949e33104beb827d06ebc83c71c8fa560128152241a52f62d4f7f6daa0`
- Balancer V2：`0xcfb9ce1213d710127eadd563ae61d58d8a35a789243ae8fee2712c05cd1110d3`
- Uniswap V2：`0xbfe5657b9758b02d679eb58d6919b181cb83c4592ded9f31d2476d973b1a6165`

这一窗口的优点在于区间较短，便于课程项目在有限资源下进行原型验证，同时三类协议样本齐全，能够检验统一框架下的多协议识别能力。

#### 25.2 真实主网验证结果

在完成真实数据预加载与识别器兼容性修正后，系统对上述区间的扫描结果表明，三笔真实样本均被成功识别：

- Aave V3 样本被识别为 `candidate = true, verified = true, strict = true`
- Balancer V2 样本被识别为 `candidate = true, verified = true, strict = true`
- Uniswap V2 样本被识别为 `candidate = true, verified = true, strict = true`

这说明当前原型系统不仅能够在本地构造样本上正常运行，而且已经在一段真实主网窗口内，对三类协议分别实现了至少一次真实样本命中。对于课程项目而言，这一结果足以支持“系统已完成原型级实现与初步真实数据验证”的结论。

#### 25.3 真实数据验证中发现的问题与修正

真实主网扫描并非一次性顺利完成，而是在运行过程中暴露了两类具有代表性的兼容性问题。

第一，`Balancer V2` 的第一版 extractor 只接受顶层 `tx.to == Vault` 的交易，因此无法识别经由其他合约间接触发 Vault flash loan 的真实样本。为此，系统后续增加了“官方 Vault `FlashLoan` event 派生 candidate”的 fallback 路径，从而覆盖了这类真实调用结构。

第二，`Aave V3` 的第一版 verifier 中，`FlashLoan` 事件的 indexed 字段定义与真实主网实现不一致，导致真实样本无法正确匹配事件。修正 ABI 与 topic 解码逻辑之后，Aave 真实样本被成功提升为 `verified strict`。

这两次修正具有方法论上的意义：它们表明仅凭本地构造样本无法充分暴露协议实现与真实链上调用模式之间的差异，而真实样本验证能够反过来帮助收紧识别规则。

#### 25.4 当前局限

尽管系统已经完成原型级实现与小窗口真实验证，但仍然存在以下局限：

- 当前真实主网验证仅覆盖少量人工确认样本，尚未进行大规模区块范围统计
- trace-backed strict verification 已实现，但仍偏向原型级调用路径验证，而非完整资金流审计
- Uniswap V2 的官方 pair 管理目前仍以预置样本与已知 pair 为主，尚未完成大规模自动发现闭环
- 目前尚未形成系统性的 precision / recall 评估，只能证明样本命中能力，不能证明全链统计精度

因此，在报告中更稳妥的表述应为：本项目完成了一个可运行的三协议闪电贷识别原型，并在本地样本与一段真实主网窗口上完成了初步验证；更大规模的链上评估与误差分析可作为后续工作继续展开。
