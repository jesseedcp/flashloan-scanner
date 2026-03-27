# 交易详情页模块说明

本文档用于逐一解释当前“交易详情页”中每个模块的作用，便于课程演示、答辩讲解和后续迭代时统一口径。

说明范围基于当前前端实现：

- 页面文件：`frontend/src/pages/TransactionDetail.tsx`
- 图形组件：`frontend/src/components/TransactionDiagrams.tsx`

## 1. 顶部摘要区

页面原文：

- 交易详情
- 区块
- 初筛命中
- 验证通过
- 严格通过
- 命中协议数
- 交互数
- 严格交互数
- 扫描结论

页面最上方展示一笔交易的核心身份信息和总判定结果。

包含内容：

- 交易哈希
- 区块号
- 是否命中初筛
- 是否验证通过
- 是否严格通过
- 命中协议数
- 交互数
- 严格交互数
- 扫描结论

作用：

- 让用户在进入详情页后第一眼知道这笔交易“是什么、发生在哪个区块、最终被判成什么级别”
- 起到总览卡的作用，避免用户一上来就进入底层 trace 细节

适合回答的问题：

- 这笔交易有没有被识别为闪电贷
- 它最终是 candidate、verified 还是 strict
- 它涉及几个协议 interaction

## 2. 扫描器结论

页面原文：

- 扫描器结论
- 最终结论
- 关键依据

这一块将扫描器已有的判定结果翻译成“人话摘要”。

包含内容：

- 最终结论标题
- 一段摘要说明
- 关键依据列表

关键依据通常来自：

- protocol_count
- interaction_count
- strict_interaction_count
- callback / repayment / settlement 命中情况
- debt opening
- exclusion reason

作用：

- 解释“为什么扫描器这样判”
- 把原始字段组织成一段可直接用于演示的结论性说明

适合回答的问题：

- 为什么这笔交易会被判为严格通过
- 扫描器做出这个结论依赖了哪些信号

## 3. 关键地址关系图

页面原文：

- 关键地址关系图
- 识别到的资产流

这一块是“谁参与了这笔交易”的结构化视图。

上半部分是关系图，下半部分是资产流列表。

### 3.1 图形部分

会展示交易中的关键地址节点，例如：

- provider
- receiver
- callback target
- factory / pair
- trace source / trace target

中间节点是当前交易本身，用来表示这些地址围绕该交易形成的交互关系。

作用：

- 帮助用户快速理解关键参与方
- 把零散地址字段变成“参与者网络”

### 3.2 识别到的资产流

这部分列出识别到的资产流记录。

通常包含：

- 动作类型，如 `transfer`、`transferFrom`、`approve`
- 资产地址
- source -> target
- amount

在 trace 可用时，优先使用 trace 提取出的 `asset_flows`。
在 trace 不可用时，则退回到 scanner summary / interaction legs。

作用：

- 补充关系图中没有展开的具体资产流信息
- 让用户看到“哪种动作、哪种资产、从哪里到哪里、数量是多少”

适合回答的问题：

- 这笔交易里有哪些关键 token 动作
- 哪些地址之间发生了资产授权或转移

## 4. 轻量时序图

页面原文：

- 轻量时序图

这一块是“交易过程怎么发生”的顺序视图。

它以泳道的形式展示：

- 参与方
- 步骤顺序
- 每一步的动作说明

trace 可用时，优先使用 `trace_summary.sequence` 生成时序图。
trace 不可用时，使用 interaction 和 legs 做 fallback。

作用：

- 帮助用户理解交易过程，而不是只看到一堆地址和字段
- 适合作为“讲故事”的模块，说明入口、借出、回调、偿还、验证等步骤

适合回答的问题：

- 这笔交易先做了什么，再做了什么
- 哪些地址在什么顺序下被调用

## 5. Trace 证据

页面原文：

- Trace 证据
- callback 命中
- 结算命中
- 归还命中
- 存在开债
- 回调帧
- 回调内部路径
- 归还路径帧
- 排除原因

这一块是 trace 层面的 interaction 证据汇总。

每个 interaction 会对应一张证据卡，通常包括：

- callback 是否命中
- settlement 是否命中
- repayment 是否命中
- 是否存在开债
- callback 帧数量
- callback 子树数量
- repayment 路径帧数量
- exclusion reason

如果 trace 拉取失败，这一块会显示失败状态和错误说明。

作用：

- 把 trace 层识别出的关键判据集中展示
- 解释严格验证、偿还路径、callback 路径等是如何被确认的

适合回答的问题：

- trace 层面是否真的看到了 callback
- trace 是否支持“已偿还”的结论
- 为什么 strict 证据成立或不成立

## 6. 完整内部调用链

页面原文：

- 完整内部调用链

这一块是最底层的 trace 展开视图。

它按调用顺序展示完整 frame 列表，每一张卡片对应一个内部调用 frame。

通常展示：

- 调用序号
- 当前 frame 的关键标题
  - token action
  - method selector
  - call type
- from -> to
- 路径标签
  - callback 子树
  - repayment 路径
  - trace error
- 补充信息
  - transfer / approve 等动作及数量
  - error / revert reason

作用：

- 提供最接近底层证据的可视化展开
- 作为“trace 已经不是摘要，而是真正展开过”的证明

适合回答的问题：

- 内部真实调用链长什么样
- 回调路径在调用链中的哪些位置
- 偿还路径在调用链中的哪些位置

注意：

- 这里展示的是 frame 列表，因此地址重复出现是正常的
- 因为同一个合约可能在多次内部调用中重复作为 caller 或 callee

## 7. 过程时间线

页面原文：

- 过程时间线

这一块是 scanner 视角的简化时间线。

它通常会按顺序展示：

- entrypoint
- asset leg
- evidence

与“轻量时序图”的区别在于：

- 轻量时序图更强调参与方之间的调用顺序
- 过程时间线更强调扫描器理解这笔交易的逻辑步骤

作用：

- 作为“中层解释层”
- 介于顶部摘要和底部原始 interaction 细节之间

适合回答的问题：

- 扫描器是如何一步步理解这笔交易的
- 哪些步骤被当成 entrypoint、asset leg 和 evidence

## 8. 关键地址

页面原文：

- 关键地址
- 地址角色

这一块是关键地址清单视图。

它会将同一个地址下的角色合并显示，例如：

- initiator
- provider
- receiver
- callback target
- repayment target
- factory
- pair

作用：

- 把重复出现的地址字段统一整理
- 帮助用户快速识别“这个地址扮演了什么角色”

适合回答的问题：

- 这笔交易里有哪些核心地址
- 每个地址分别承担什么角色

## 9. 验证证据

页面原文：

- 验证证据
- 方法选择器
- 排除原因
- 验证说明

这一块是 interaction 级别的验证卡片。

通常展示：

- callbackSeen
- settlementSeen
- repaymentSeen
- containsDebtOpening
- method selector
- exclusion reason
- verification notes

和“Trace 证据”的区别在于：

- `Trace 证据` 更强调 trace summary 提取出的证据
- `验证证据` 更强调 interaction 自身的验证字段和扫描器说明

作用：

- 作为 interaction 层面的验证说明
- 帮助用户理解“这条 interaction 为什么被判通过/不通过”

适合回答的问题：

- 某个 interaction 的验证字段到底是什么状态
- 为什么 interaction 不是 strict 或为什么被排除

## 10. 原始 Interaction 明细

页面原文：

- 每个协议 interaction 模块
- candidate level
- verified / strict
- provider / receiver / callback target
- initiator / on_behalf_of
- factory / pair

这是详情页最底部的“原始底稿区”。

页面会按 interaction 逐块展示：

- 协议
- entrypoint
- candidate level
- verified / strict
- provider / receiver / callback target
- initiator / on_behalf_of
- factory / pair
- 其他 interaction 字段

作用：

- 保留原始结果结构
- 作为前面所有摘要模块的“证据底稿”

适合回答的问题：

- 如果我要查最原始的 interaction 数据，应该看哪里
- 顶部结论与中间图表分别对应到底层哪组字段

## 11. 资产流明细

页面原文：

- 资产流
- asset address
- asset role
- amount borrowed
- amount repaid
- premium / fee
- strict leg
- event seen
- settlement mode

在每个 interaction 模块下面，还会显示资产流（asset flow / legs）部分。

通常包括：

- asset address
- asset role
- amount borrowed
- amount repaid
- premium / fee
- strict leg
- event seen
- settlement mode

作用：

- 展示资产维度上的细节
- 帮助用户理解每条 leg 对应的借出、归还和费用关系

适合回答的问题：

- 借的是什么资产
- 还了多少
- 有没有 premium / fee
- 哪条 leg 被视为 strict leg

## 12. 模块之间的阅读顺序建议

建议从上到下这样理解：

1. 先看顶部摘要，确认这笔交易的最终结论
2. 再看扫描器结论，理解为什么会得出这个结果
3. 看关键地址关系图与轻量时序图，理解参与方和交易过程
4. 看 Trace 证据和完整内部调用链，验证底层证据是否充分
5. 看过程时间线、关键地址、验证证据，理解中层分析结构
6. 最后看原始 interaction 明细和资产流明细，回到底稿

这也是当前详情页的设计目标：

- 上半部分偏“分析解释”
- 下半部分偏“证据底稿”

## 13. 当前页面的整体定位

当前交易详情页不是一个纯字段页，也不是完整自动报告系统。

它当前的定位是：

- 基于 scanner 结果和 trace summary 的“轻量分析页”
- 既能支持演示讲解，也保留底层证据

因此它同时承担三种职责：

- 总览：告诉用户这笔交易是什么
- 分析：解释为什么被判定为闪电贷
- 证据：展示 interaction、legs 和 trace frame 作为底稿
