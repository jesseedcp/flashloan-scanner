# 交易详情页增强方案

> 文档角色：详情页增强方案  
> 目标读者：继续实现该项目的工程同学 / AI  
> 文档用途：明确当前交易详情页还能增强哪些信息、哪些可以直接做、哪些需要补后端逻辑、哪些第一版不建议做

## 1. 背景

当前详情页已经可以展示：

- `tx hash`
- `block number`
- candidate / verified / strict
- interaction 列表
- leg 列表
- verification notes

但它更像“字段页”，还不是“分析页”。

用户点进详情后，虽然能看到很多字段，却不容易快速回答下面这些问题：

1. 这笔交易到底是什么？
2. 涉及了哪些关键地址？
3. 借了什么、还了什么？
4. 为什么扫描器判定它是 candidate / verified / strict？
5. 最终应该如何用一句话总结这笔交易？

因此，详情页下一阶段的目标不是继续堆字段，而是把现有数据组织成更接近“分析报告”的结构。

## 2. 设计目标

增强后的详情页应优先回答这 5 个问题：

### 2.1 交易概览

- 交易哈希
- 区块号
- 命中协议
- interaction 数
- 是否 candidate / verified / strict

### 2.2 关键地址

- 谁发起了这笔交互
- 谁接收了回调
- 谁提供了流动性
- 哪些地址是 pair / factory / provider / repayment target

### 2.3 资产流

- 每条资产腿借出了什么
- 归还了什么
- premium / fee 是多少
- 哪条 leg 是 strict leg

### 2.4 验证证据

- callback 是否命中
- settlement 是否命中
- repayment 是否命中
- 是否存在 debt opening
- exclusion reason / verification notes 是什么

### 2.5 扫描器结论

- 为什么被判定为真实闪电贷
- 为什么是 strict 或 non-strict
- 如果不是 strict，卡在什么地方

## 3. 当前已有数据

现有 `GET /api/v1/transactions/:txHash` 已经返回的核心字段，定义在：

- [query_service.go](/D:/Users/dd/Desktop/5566project/api/service/query_service.go)

当前 `TransactionDetailResponse` 已包含：

- 交易层：
  - `tx_hash`
  - `chain_id`
  - `block_number`
  - `candidate`
  - `verified`
  - `strict`
  - `interaction_count`
  - `strict_interaction_count`
  - `protocol_count`
  - `protocols`

- interaction 层：
  - `interaction_id`
  - `protocol`
  - `entrypoint`
  - `provider_address`
  - `factory_address`
  - `pair_address`
  - `receiver_address`
  - `callback_target`
  - `initiator`
  - `on_behalf_of`
  - `candidate_level`
  - `verified`
  - `strict`
  - `callback_seen`
  - `settlement_seen`
  - `repayment_seen`
  - `contains_debt_opening`
  - `exclusion_reason`
  - `verification_notes`
  - `raw_method_selector`

- leg 层：
  - `leg_index`
  - `asset_address`
  - `asset_role`
  - `token_side`
  - `amount_out`
  - `amount_in`
  - `amount_borrowed`
  - `amount_repaid`
  - `premium_amount`
  - `fee_amount`
  - `interest_rate_mode`
  - `repaid_to_address`
  - `opened_debt`
  - `strict_leg`
  - `event_seen`
  - `settlement_mode`

这说明：详情页的“信息密度”已经不低，问题主要是组织方式，而不是完全没有数据。

## 4. 可以直接做的部分

下面这些能力，基本不需要增加新的链上解析逻辑，主要靠前端重组现有字段，或后端做很薄的一层 DTO 整理即可。

### 4.1 顶部摘要卡

可以直接做。

建议展示：

- 交易哈希
- 区块号
- 协议数
- interaction 数
- strict interaction 数
- candidate / verified / strict

原因：

- 这些字段已经在 `TransactionDetailResponse` 顶层存在
- 不需要新增 DB 查询

### 4.2 关键地址分组

可以直接做。

建议将地址按角色分组显示：

- 发起者：`initiator`
- 代偿关系：`on_behalf_of`
- 回调接收者：`receiver_address`
- 回调目标：`callback_target`
- 流动性来源：`provider_address`
- 工厂 / Pair：`factory_address` / `pair_address`
- 归还目标：`repaid_to_address`

说明：

- 这不是“链上标签识别”
- 这是“基于扫描器字段的角色分组”
- 非常适合第一版演示

### 4.3 资产流表重排

可以直接做。

建议把现有 `legs` 从散字段改成更清晰的表：

- leg index
- asset address
- asset role
- borrowed
- repaid
- premium
- fee
- opened debt
- strict leg
- event seen

说明：

- 数据已完整存在
- 只是需要前端按分析视图重新排版

### 4.4 验证证据卡

可以直接做。

每个 interaction 可以展示一组证据项：

- callback seen
- settlement seen
- repayment seen
- contains debt opening
- strict
- exclusion reason
- verification notes

说明：

- 这部分是最适合答辩展示的内容之一
- 因为它直接解释“为什么系统这么判”

### 4.5 扫描器结论摘要

可以直接做“规则总结版”。

建议根据已有字段生成人话摘要，例如：

- 该交易命中了 `Aave V3` 的闪电贷交互
- 检测到 `1` 条严格 interaction
- callback 与 repayment 证据完整
- 不存在 debt opening
- 因此被判定为 strict 样本

说明：

- 这不是大模型分析
- 只是后端或前端基于固定规则生成摘要
- 成本低，但效果很好

### 4.6 轻量时间线

可以直接做“摘要时间线版”。

建议时间线展示：

1. 进入某协议入口 `entrypoint`
2. 借出哪几条资产腿
3. 回调执行
4. 偿还资产
5. 扫描器给出结论

说明：

- 这不是 EVM trace 级别的 sequence diagram
- 但已经足够形成“过程叙事”

## 5. 需要一些后端拼装逻辑的部分

这些功能不是不能做，但建议由后端新增一个更适合详情页消费的 summary DTO，避免前端写太多推导逻辑。

### 5.1 地址标签归类

可以做“规则归类版”，但最好后端来统一整理。

原因：

- 现有地址字段分散在 interaction 和 leg 中
- 同一地址可能扮演多种角色
- 前端自己拼会比较乱

建议新增 DTO：

```go
type AddressRoleSummary struct {
    Address string   `json:"address"`
    Roles   []string `json:"roles"`
}
```

用途：

- 把重复地址合并
- 给每个地址汇总角色标签

### 5.2 资金流向图

可以做“轻量关系图版”，但需要后端先整理节点和边。

建议后端输出：

```go
type FlowNode struct {
    ID    string `json:"id"`
    Label string `json:"label"`
    Kind  string `json:"kind"`
}

type FlowEdge struct {
    From        string `json:"from"`
    To          string `json:"to"`
    Asset       string `json:"asset"`
    Borrowed    string `json:"borrowed"`
    Repaid      string `json:"repaid"`
    Premium     string `json:"premium"`
    Fee         string `json:"fee"`
    Description string `json:"description"`
}
```

注意边界：

- 这不是完整 token transfer 图
- 只能表达“扫描器识别到的闪电贷交互关系”

### 5.3 利润 / 成本分析

可以做“扫描视角估算版”，但要明确口径有限。

后端可以整理：

- 每条 leg 的 borrowed / repaid / premium / fee
- 每个 asset 的净变化
- 全交易的闪电贷成本摘要

不能直接承诺：

- 最终盈利是多少 USD
- 攻击者真实净利润是多少

因为目前没有：

- token decimals / symbol 统一口径
- 实时价格
- 全交易 transfer 明细
- 外部收益来源

### 5.4 攻击路径总结

可以做，但最好后端统一生成文本或结构化 conclusion。

原因：

- 这是一种规则归纳
- 放在后端更稳定，也更容易复用到导出报告或 API

建议新增：

```go
type DetectionConclusion struct {
    Summary      string   `json:"summary"`
    KeyReasons   []string `json:"key_reasons"`
    WeakSignals  []string `json:"weak_signals"`
    FinalVerdict string   `json:"final_verdict"`
}
```

## 6. 第一版不建议做的部分

下面这些功能如果硬做，会让项目快速从“演示控制台”扩展成“链上报告生成器”，第一版不建议投入。

### 6.1 完整 sequence diagram

不建议第一版做。

原因：

- 当前数据不是完整 trace 调用树
- 没有严格的 call-level 时序
- 做出来的图会像真的，但证据链不够扎实

第一版更建议做“摘要时间线”，不要做严格泳道时序图。

### 6.2 完整资金流审计图

不建议第一版做。

原因：

- 当前没有完整 transfer graph
- 没有 token symbol / decimals 全量补充
- 容易误导成“全交易已经被完全还原”

### 6.3 精确利润分析

不建议第一版做。

原因：

- 当前最多能做“闪电贷成本分析”
- 不能严肃承诺“真实利润”

## 7. 推荐的详情页结构

建议增强版详情页按下面顺序组织：

### 7.1 顶部摘要

- tx hash
- block
- 协议
- interaction 数
- strict interaction 数
- candidate / verified / strict

### 7.2 关键地址

- 按角色分组列出所有关键地址

### 7.3 资产流

- 以 leg 为核心的表格

### 7.4 验证证据

- 每个 interaction 的证据项

### 7.5 过程时间线

- 轻量时间线，不做完整 trace 图

### 7.6 扫描器结论

- 用人话总结最终结论

## 8. 推荐实施顺序

建议按下面顺序推进：

1. 顶部摘要卡
2. 关键地址区
3. 资产流表重排
4. 验证证据卡
5. 扫描器结论摘要
6. 轻量时间线
7. 最后再评估是否需要关系图

## 9. 最适合第一版演示的增强点

如果只选 3 个最值得做的点，建议优先做：

### 9.1 关键地址分组

原因：

- 一眼能看懂交易涉及哪些地址
- 比现在字段散列更像分析页面

### 9.2 验证证据卡

原因：

- 最能解释“为什么系统这么判”
- 对答辩最有帮助

### 9.3 扫描器结论摘要

原因：

- 能把字段串成叙事
- 从“数据页”变成“分析页”

## 10. 一句话结论

基于当前已有数据，详情页完全可以升级成“半报告化分析页”。

第一版最合理的边界是：

- 不做完整攻击分析平台
- 不做完整 trace 报告
- 重点把已有字段重组为“摘要 + 地址 + 资产流 + 证据 + 结论”

这样既能明显提升演示效果，也不会把工程范围扩得失控。
