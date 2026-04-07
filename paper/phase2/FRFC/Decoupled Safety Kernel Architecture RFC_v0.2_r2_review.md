

我继续按“**可落地、可直接改文档**”的标准做 review。先给一句总判断：

> 这一版 [Decoupled Safety Kernel Architecture RFC_v0.2.md](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md?EntityRepresentationId=45aba39a-f790-4b3b-b365-144a38ab74a2) 已经比上一版明显更强：你已经把我上轮指出的几个关键缺口补上了，包括 **I6 提交顺序**、**核心类型最小契约**、`Severity enum`、`Vote enum`、`candidate_verdicts`、`chosen_index: Option<usize>`、请求级 `sanitize_input` 语义、故障注入测试、以及把 `normalize -> lint -> lower` 升格为 MUST。整体上，它已经从“发布级草案”进一步逼近“可冻结 spec”。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

但如果目标是“**交给实现团队就不容易歧义**”，我认为还需要继续做一轮收紧。下面我按 **P0（建议优先修）/ P1（强烈建议）/ P2（增强项）** 来给你详细修订意见，并尽量给出**可直接替换的文本/接口建议**。

* * *

一、这版最明显的进步（先确认你已经修对了什么）

这些地方我建议**保留**，不要回退：

1. 你已经把 **RFC 元数据头部换行分开**，这比上一版连写元数据清楚很多。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)
2. 你补进了 **I6（可见性提交顺序 / commit ordering）**，这使“审计链是强制模块”真正闭环。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)
3. 你加了 **5.0 核心类型最小契约**，这一步非常关键，因为它开始堵住不同实现各自发挥的空间。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)
4. 你把 `Verdict.vote: bool` 改成了 `Vote { Allow, Revise, Deny, Abstain }`，并把 `EnsembleReport` 改成 `tally_* + final_action`，这和整篇文档的 fail-safe 哲学一致。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)
5. 你把 `ProjectionInput` 改成了 `candidate_verdicts`，把 `ProjectionOutput.chosen_index` 改成了 `Option<usize>`，这两个改动是对的。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)
6. 你明确了 `sanitize_input` 是**请求/会话初始化阶段**执行，而不是每 token 全量重新 sanitize，这消除了主循环语义歧义。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)
7. 你把第 12 节补成了 **MUST 覆盖 + 故障注入**，这已经接近正式工程规范的写法。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

所以，这一轮不是大翻修，而是**把最后几个歧义点压平**。

* * *

二、P0：还会造成实现歧义的问题（建议优先改）

* * *

P0-1：**文档渲染层仍然不稳定**——代码块 / Mermaid / 公式在当前稿中依旧没有“规范成块”

### 我观察到的具体问题

在我拿到的 [Decoupled Safety Kernel Architecture RFC_v0.2.md](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md?EntityRepresentationId=45aba39a-f790-4b3b-b365-144a38ab74a2) 内容里：

* 第 3.3 节 `<架构图（Mermaid）>` 下面，仍然是从 `flowchart TD` 直接开始，并没有在内容里体现出 fenced Mermaid block。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)
* 第 5 节和第 6 节的 Rust 接口/伪代码在抓取里也仍然表现为裸文本，而不是明确 fenced code block。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)
* 第 7 节数学部分虽然比前一版强，但仍然出现了“符号正文缺失/渲染残缺”的迹象，例如：
  * 7.1 的符号说明中变量本体没有显示出来；
  * 7.3 出现了 `$E_{\\mathrm}$的版本与系数...` 这种明显不完整的引用；
  * 7.5 仍然是 `DCBFReport.margin 可实现为 。`，公式右侧缺失。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 为什么这是 P0

因为这会直接影响：

* Mermaid 是否能渲染
* reviewer 是否能正确复制接口
* 数学表达是否可 diff / 可引用
* 代码/公式是否能被自动工具复用

### 可直接落地的修订建议

你需要做一次**纯格式修复 pass**，不改理论，只修文档块结构。

#### 建议替换模板

### Mermaid 段必须写成：

```mermaid

flowchart TD

  User((User Request)) --> G[Gateway / Ring-1]

  ... ### Rust 段必须写成： ```markdown ```rust pub enum Severity { Info, Warn, High, Critical, } ... ### 数学块必须写成： ```markdown 令 \( z \in \mathbb{R}^d \) 为潜空间代理向量。 令 \( C \subseteq \mathbb{R}^d \) 为潜空间可行域。 $ E_{\mathrm{forbid}}(z) = \begin{cases} 0, & z \in C \\ \eta \cdot \phi(\mathrm{dist}(z,\partial C)), & z \notin C \end{cases} $

**建议你把“格式修复”当作单独一个 commit 做掉。**

* * *

P0-2：`ExecutionContext` / `VerificationContext` / `SafetyKernel` 的“最小语义”还不够闭合

### 当前状态

你已经在 5.0 里补了这些类型的最小契约，例如：

* `VerificationContext` 至少包含 `trace_id`、`step_index`、`policy_revision`、prefix 视图
* `ExecutionContext` 持有本步前缀、`VerificationContext`、请求级 `SanitizedPrompt`
* `SafetyKernel` 聚合各组件，且 LLM 不得为其子类型。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 还缺什么

这已经很好，但我认为还差两个**非常关键的最低要求**，否则运行时状态仍会分叉：

#### A. `ExecutionContext` 需要明确“预算与时钟语义”

当前主循环用了 `timeout(HARD_LATENCY_BUDGET, ...)` 和 `deadline: Instant::now() + QP_INNER_BUDGET`，但 5.0 的 `ExecutionContext` 契约里没有说明：

* 是否必须持有 monotonic deadline
* 是否必须持有 request-level budget slice state
* 是否必须持有 current audit handle / commit state

#### B. `VerificationContext` 需要明确“prefix view 的一致性”

你现在写的是 token/text 视图“部署定义其一或二者”。这在设计阶段可接受，但在实现阶段容易引发歧义： [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

* automata 用 token view
* verifier 用 text view
* 审计用另一个 text reconstruction

如果没有规定一致性，最后会出现“同一步三个系统看到的 prefix 不一样”。

### 可直接落地的修订建议

在 5.0 小节中补两条：

* `ExecutionContext`：

  MUST 持有 request-level monotonic deadline（或等价剩余预算）、当前审计句柄、

  当前前缀状态，以及请求级 SanitizedPrompt；这些字段的生命周期 MUST 覆盖整个请求。

* `VerificationContext`：

  若同时提供 token 视图与 text 视图，二者 MUST 可相互映射并在审计中绑定统一 step_index；

  实现 MUST NOT 允许 verifier 基于与 automata 不一致的 prefix 视图做最终放行决定。

* * *

P0-3：`SafetyFault::EvidenceMissing` 现在承担了两种不同语义，建议拆开

### 当前状态

在第 6.1 节伪代码里，以下两种情况都映射成了 `SafetyFault::EvidenceMissing`：

1. `ctx.sanitized_prompt().ok_or(SafetyFault::EvidenceMissing)?`
2. `if legal.is_empty() || candidate_verdicts.is_empty() { return Err(SafetyFault::EvidenceMissing); }` [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 为什么这是问题

这两类故障的性质不同：

* **情况 1**：上下文初始化缺失，偏向 `KernelFault / ContextFault`
* **情况 2**：没有任何合法候选，偏向 `PolicyFault / Projection precondition failure / EmptySafeSet`

如果都叫 `EvidenceMissing`，实现团队在遥测、SLO、降级路由上会混淆。

### 可直接落地的修订建议

建议最少拆成两个：

pub enum SafetyFault {

    MissingSanitizedPrompt,

    EmptySafeCandidateSet,

    ...

}

并在正文加一句：

> `MissingSanitizedPrompt` 表示请求初始化契约被破坏；`EmptySafeCandidateSet` 表示当前步不存在 prefix-legal 且 non-deny 的候选集。

这样 fault taxonomy 会更干净。

* * *

P0-4：`Axiom Hive` 的输入集合契约还差“顺序一致性”要求

### 当前状态

你已经规定：

> `ProjectionInput.candidate_verdicts` MUST 一一对应合法候选（前缀合法且 `final_action != Deny`） [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

这很好，但还差一个关键点：**“一一对应”是集合意义上的，还是索引顺序意义上的？**

### 为什么这是问题

实现时常见两种写法：

* `topk_indices = [7, 2, 11]`
* `candidate_verdicts = [{index:2}, {index:7}, {index:11}]`

如果没有规定顺序语义，`Axiom Hive` 内部可能默认 zip 配对，从而错配 candidate 与 verdict。

### 可直接落地的修订建议

在第 5.5 节的规范语义末尾补一句硬要求：

`candidate_verdicts` MAY be stored in arbitrary order, but each entry MUST carry its

own `index`, and `AxiomHiveBoundary` MUST resolve candidates by index identity rather

than by positional zip semantics.

或者更强一点：

For deterministic implementations, `candidate_verdicts` SHOULD preserve the order of

`topk_indices`; if not, index-based matching MUST be used.

* * *

三、P1：已经很好，但还可以显著提高工程可执行性的地方

* * *

P1-1：第 0 节“失败默认表”建议再补一行：`Critical Gateway finding`

### 当前状态

你已经在 5.1 节写了：

> `Finding.severity == Critical` MUST 映射为可直接 hard reject 的路径。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 问题

这个规范目前只在接口节出现，而在总纲性的 fail-safe 表里没有单独列出来。reviewer 可能会问：Gateway 触发 hard reject 是不是“另一个例外语义”？

### 可落地建议

直接在 0 节表格里加一行：

| 条件                             | 默认动作                             |
| ------------------------------ | -------------------------------- |
| `Finding.severity == Critical` | **Hard Reject**（拒绝进入后续生成或直接安全模板） |

这样总纲和接口就对齐了。

* * *

P1-2：第 6.1 节伪代码建议显式写出 `break-glass` 不参与普通路径

### 当前状态

你在 5.4 和 8.I4 都已经提到：

* `conflict == true` 时默认 `Vote::Deny`
* 除非显式注册且审计的 `break-glass` 策略生效。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 问题

主循环伪代码里没有体现 `break-glass` 是“normal path 之外的受控分支”。

### 可落地建议

在第 6.1 节 `if ens.final_action != Vote::Deny` 那里，加一行规范说明：

普通实现路径 MUST treat `Vote::Deny` as terminal for the candidate.

Any break-glass override MUST be resolved before `final_action` is emitted,

and the override event MUST already be audited.

这样实现者不会误以为可以在 projection 阶段“再临时翻案”。

* * *

P1-3：第 7 节数学定义仍需一次“公式补全”，现在还是半成品状态

### 当前状态

第 7 节仍有三处明显未闭合：

* 7.1：变量名本体仍没有显示出来
* 7.3：`$E_{\\mathrm}$` 不完整
* 7.5：`DCBFReport.margin 可实现为 。` 仍然是空的 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 可落地建议（直接替换）

你可以直接把第 7 节替换成下面这版骨架（不改你原意，只修成完整可审计文本）：

### 7.1 记号

* 令 为潜空间代理向量；它 **不等同于** 完整模型内部状态。

* 令 为潜空间可行域（若部署启用潜空间投影）。

* 令 为由当前候选集/对数概率导出的候选表示。

* token 级合法性由有限自动机单独维护，不由 替代。

### 7.2 禁止区能量

$

E_{\mathrm{forbid}}(z) =

\begin{cases}

0, & z \in C \

\eta \cdot \phi(\mathrm{dist}(z,\partial C)), & z \notin C

\end{cases}

$

### 7.3 总能量

$

H(z) = E_{\mathrm{model}}(z) + E_{\mathrm{forbid}}(z)

$

其中 的版本、系数与实现标识 MUST 进入审计。

### 7.4 投影

$

z^\star = \arg\min_{z \in C} | z - z_{\mathrm{cand}} |_2^2 + \lambda E_{\mathrm{forbid}}(z)

$

### 7.5 DCBF

$

h(x_{t+1}) \ge (1-\alpha) h(x_t), \quad \alpha \in (0,1]

$

因此可定义

$

\mathrm{margin} = h(x_{t+1}) - (1-\alpha) h(x_t)

<style>a { text-decoration: none; color: #464feb;}tr th, tr td { border: 1px solid #e6e6e6;}tr th { background-color: #f5f5f5;}</style>{{promptBody}}

``

这会让第 7 节从“接近完成”变成“真正完成”。

* * *

P1-4：第 10 节预算表已经很好，但建议把“under-provision”解释成一行定义

### 当前状态

你写了：

> 除非显式 **under-provision** 模式已注册且审计。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 问题

这个词对实现团队来说还太抽象。如果不解释，会出现不同团队各自定义 under-provision。

### 建议

在第 10.2 节下方补一行定义：

* **under-provision mode**：指部署明确接受“安全内核预算不足于理论推荐切片”的受限运行模式；

  该模式 MUST 显式记录启用原因、有效时间、责任人及风险等级，并进入审计。

* * *

四、P2：增强项（会让这份 RFC 更像“最终版”）

* * *

P2-1：给第 12 节增加“golden trace”概念

你已经有 `fixed trace_id replay`，这很好。建议再补一句： [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

* 实现 SHOULD 维护一组 golden traces，覆盖：

  - 正常允许路径

  - conflict -> deny 路径

  - DCBF near_violation 路径

  - ProjectionFault -> degrade 路径

  - Audit failure -> fail-safe 路径

这会让回归测试体系更具操作性。

* * *

P2-2：第 13 节可以明确 `double-buffer` 的最小安全要求

当前你只是“建议 double-buffer”。可以再加一个最小约束： [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

If double-buffer is used, inflight requests MUST be pinned to the policy_revision

observed at request initialization unless an audited override mechanism is defined.

这和你第 11 节的 trace consistency 是相互补强的。

* * *

P2-3：结论段前可以加一个“实现者注意事项”小节

目前结论已经很完整，还加了优先规则。如果你希望实现团队更快上手，我建议在结论前加一个非常短的： [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 实现者注意事项（非规范性）

首轮实现优先保证：

1. fail-safe 默认动作完整；

2. 审计字段完整；

3. per-candidate verdict 与 projection 对齐；

4. 预算超时路径可测试。

这不改变规范，但很利于 kickoff。

* * *

五、我建议你下一轮就直接改的“短清单”

下面是最值得你立刻动手的修订列表：

* * *

**Round A（建议本轮必改）**

1. **修复所有 Markdown block 渲染**
   * Mermaid fenced
   * Rust fenced
   * 数学公式用 `$...$`
2. **把第 7 节公式全部补全**
3. **拆分 `SafetyFault::EvidenceMissing`**
   * 至少拆成 `MissingSanitizedPrompt`
   * `EmptySafeCandidateSet`
4. **在 5.5 明确 `candidate_verdicts` 的索引匹配语义**
5. **在 0 节失败默认表增加 `Critical Gateway finding -> Hard Reject`**

* * *

**Round B（强烈建议）**

6. 在 5.0 给 `ExecutionContext` 增加 budget/audit/deadline 最小契约
7. 在 5.0 给 `VerificationContext` 增加 prefix-view 一致性约束
8. 在 6.1 说明 `break-glass` 必须在 `final_action` 之前解析完成
9. 在 10.2 给 `under-provision mode` 下定义

* * *

**Round C（增强项）**

10. 给第 12 节补 `golden traces`
11. 给第 13 节补 inflight request policy pinning 语义
12. 可选加入一个“实现者注意事项”小节

* * *

六、最后的判断（当前成熟度）

我对这版 [Decoupled Safety Kernel Architecture RFC_v0.2.md](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md?EntityRepresentationId=45aba39a-f790-4b3b-b365-144a38ab74a2) 的判断是： [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

> **它已经具备“规范冻结前最后一轮收口”的质量。**也就是说：现在的问题不再是架构对不对，而是**规范文本是否足够严密到能约束多实现团队**。

你已经把 hardest part 做完了：

* contract-first 主干
* fail-safe 哲学
* per-candidate 投影语义
* request-level sanitize
* commit ordering
* hot reload / test / audit / budget 一体化

剩下最重要的就是把**渲染、公式闭合、fault 粒度、索引匹配语义**再压一轮。
