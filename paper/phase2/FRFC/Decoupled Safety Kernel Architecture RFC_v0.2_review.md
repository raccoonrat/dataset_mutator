我继续按“**可落地、可直接改文档**”的标准来 review。先给总判断：

> 这版 [Decoupled Safety Kernel Architecture RFC_v0.2.md](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md?EntityRepresentationId=04fad6b2-9f02-4a7f-9ccf-793321cc9fa4) 已经明显进入“**发布级整合稿**”的形态：你补上了摘要、RFC 2119、非目标、威胁模型、安全不变量、故障分类、预算/审计/测试/热更新/兼容性这些治理层章节，而且保留了 contract-first 主干、三类约束分解、厚接口、top-k 主循环和文本化数学定义。整体方向是对的。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

但如果你现在要把它作为**真正给架构委员会/实现团队评审的 RFC**，我认为还有几类问题必须继续收紧。下面我按 **P0（阻塞发布）/ P1（强烈建议）/ P2（增强项）** 给出详细、可执行修订意见。

* * *

一、P0：阻塞发布的问题（建议先改）

* * *

P0-1：文档渲染层还有明显“格式破损”，会直接影响评审质量

### 问题

在 [Decoupled Safety Kernel Architecture RFC_v0.2.md](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md?EntityRepresentationId=04fad6b2-9f02-4a7f-9ccf-793321cc9fa4) 里，头部 metadata 仍然是连写的，例如：

* `**Author:** Anthropic Claude Code**Supersedes:** ...**Incorporates:** ...`
* 摘要后面的“**非目标**不在摘要展开...”等也与正文没有明显段落隔离。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

同样地，Mermaid 段和 Rust 接口段在当前抓取结果里表现为**没有代码围栏**的裸文本；伪代码和数学公式也存在若干行内/块级 LaTeX 混杂、转义不稳定的问题。比如第 3.3 节 Mermaid 图直接从 `flowchart TD` 开始，第 5 节和第 6 节的 Rust 代码段也没有在结果中表现出稳定的 fenced code block。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 为什么这是 P0

这不是“美观问题”，而是**规范文本可读性与可审计性问题**。一旦 Markdown 渲染不稳定，后果包括：

* Mermaid 不能直接渲染
* Rust 接口无法被 reviewer 快速扫描
* 数学公式 diff 困难
* 实现团队复制代码时容易引入误差

### 具体修订建议

**必须统一全文的 Markdown block discipline：**

#### 1）头部 metadata 改成列表，不要行内连写

建议直接改成：

# RFC: Decoupled Safety Kernel Architecture (v0.2)

**Author:** Anthropic Claude Code 

**Supersedes:** Decoupled Safety Kernel Architecture RFC_v0.1.md（contract-first 修订版） 

**Incorporates:** Decoupled Safety Kernel Architecture RFC_v0.2_review.md（治理层与完整性补强） 

**Target audience:** 架构委员会、安全评审、内核/系统实现团队 

**Status:** Draft — 发布级整合稿

#### 2）所有 Mermaid 图必须强制 fenced

```mermaid

flowchart TD

... #### 3）所有 Rust/伪代码必须 fenced ```markdown ```rust ... #### 4）所有块级公式必须统一用 `$ ... $` 不要混用裸 LaTeX、半转义公式、HTML 与 Markdown。 --- ## P0-2：接口契约虽然变厚了，但**仍缺“最小可实现类型闭包”** ### 问题 [Decoupled Safety Kernel Architecture RFC_v0.2.md](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md?EntityRepresentationId=04fad6b2-9f02-4a7f-9ccf-793321cc9fa4) 第 5 节接口已经比早期版本强很多，但仍然有一批**关键类型没有被定义或没有最小语义约束**，例如： - `Ast` - `Token` - `LatentState` - `VerificationContext` - `DeterministicAutomaton` - `CompileError` - `SystemFault` - `SafetyFault` - `ExecutionContext` - `SafetyKernel` - `SafeToken` ### 为什么这是 P0 这会导致 RFC 在实现交接时落入“大家都懂，但每个人实现得不一样”的状态。 当前接口已经足够像规范，但**缺少最小类型词典（minimal type contract）**，所以还不是完整的规范。 ### 具体修订建议 在第 5 节前加一个 **“5.0 术语与核心类型最小定义”**，不需要全量实现，但至少要规定： ```markdown ### 5.0 核心类型最小契约（Normative Type Surface） - `Token`：MUST 表示模型可见的最小离散输出单位；实现 MAY 为 token id、byte-pair token 或 grammar symbol，但 MUST 可映射回审计可见文本片段。 - `LatentState`：MUST 表示只读代理状态，而非完整模型内部状态；MUST 声明其来源层、维度与 probe 版本。 - `DeterministicAutomaton`：MUST 表示可增量验证 prefix legality 的可执行工件；MUST 支持状态序列化摘要进入审计。 - `VerificationContext`：MUST 至少包含 trace_id、step_index、policy_revision、prefix_text/token view。 - `SafetyFault`：MUST 可区分 GatewayFault / MonitorFault / PolicyFault / VerifierFault / ProjectionFault / KernelFault。 - `SafeToken`：MUST 表示最终对外可见输出单元；若为降级输出，MUST 可追溯其来源动作（Template/Refuse/Redact）。

这样你的接口就从“看上去完整”变成“实现上可对齐”。

* * *

P0-3：`Judge Ensemble` 的投票类型仍然过弱，跟整篇文档的裁决哲学不一致

### 问题

[Decoupled Safety Kernel Architecture RFC_v0.2.md](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md?EntityRepresentationId=04fad6b2-9f02-4a7f-9ccf-793321cc9fa4) 第 5.4 节里，`Verdict` 仍然是：

pub struct Verdict {

pub vote: bool,

...

}

而 `EnsembleReport` 则只记录 `tally_pass / tally_fail / conflict / final_allow`。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 为什么这是 P0

你在第 0 节、第 4 节和第 8 节里其实已经把系统哲学写得很清楚了：

* 冲突默认 deny
* 证据不足默认 deny
* 输出合法性不应坍缩成单布尔
* Degradation 有多种动作 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

但这里 `vote: bool` 仍然把整个 verifier 层压扁成了“过/不过”。这会直接造成两个问题：

1. `Revise` 和 `Abstain` 没有一等地位
2. Axiom Hive 只能知道“过没过”，不知道是“可修正”还是“必须拒绝”

### 具体修订建议

把 `bool` 改成显式枚举，并把计票规则写成规范，而不是留给实现猜：

pub enum Vote {

    Allow,

    Revise,

    Deny,

    Abstain,

}

pub struct Verdict {

    pub vote: Vote,

    pub confidence: f32,

    pub explanation: String,

    pub verifier_id: String,

}

pub struct EnsembleReport {

    pub verdicts: Vec<Verdict>,

    pub tally_allow: u32,

    pub tally_revise: u32,

    pub tally_deny: u32,

    pub tally_abstain: u32,

    pub conflict: bool,

    pub final_action: Vote, // MUST obey default-deny policy

}

``

并在正文增加一句硬约束：

> `final_action` MUST be derived by an explicit vote policy; conflict MUST resolve to `Deny` unless a pre-registered break-glass policy is active.

* * *

P0-4：主循环里 `EnsembleReport` 的聚合仍然是“未定义行为”

### 问题

你在第 6.1 节伪代码里已经很诚实地写了注释：

> “实现 MUST 将 EnsembleReport 聚合策略（如按 logit 选最优对应报告、worst-case 合并等）写入审计；上文‘最后一个合法候选’仅为占位示例。” [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

也就是说，当前主循环实际上仍然存在一个**规范空洞**：

* `legal: Vec<usize>` 是整个合法候选集
* 但 `ensemble_for_projection: Option<EnsembleReport>` 只保留了一个候选的报告
* 最后 Axiom Hive 用的是这个单一报告，而不是“合法候选集的可对齐裁决摘要” [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 为什么这是 P0

这是**真正的 contract leak**。Axiom Hive 的输入已经被你定义成候选集联合投影，但裁决摘要却仍然是“单候选占位”。这个矛盾会直接让实现团队无法判断：

* projection 是在“所有 legal 候选的共同约束”上做
* 还是在“某一个候选的 ensemble 结果”上做
* 冲突是 per-candidate 还是 per-step

### 具体修订建议

你必须在 RFC 里**选定一种规范语义**。我建议用最稳妥的方案：方案 A（推荐）：**Per-candidate verdict map**

也就是把 `ProjectionInput` 改成：

pub struct CandidateVerdict {

    pub index: usize,

    pub ensemble: EnsembleReport,

}

pub struct ProjectionInput<'a> {

    pub logits: &'a [f32],

    pub topk_indices: &'a [usize],

    pub candidate_verdicts: &'a [CandidateVerdict],

    pub automata: &'a DeterministicAutomaton,

    pub dcbf: &'a DCBFReport,

    pub deadline: std::time::Instant,

}

然后正文明确：

> Axiom Hive MUST operate on the subset of candidates that are prefix-legal and whose per-candidate `EnsembleReport.final_action != Deny`.

这样主循环和接口就闭环了。

* * *

P0-5：`ProjectionOutput.chosen_index: usize` 在 infeasible 场景下语义不安全

### 问题

[Decoupled Safety Kernel Architecture RFC_v0.2.md](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md?EntityRepresentationId=04fad6b2-9f02-4a7f-9ccf-793321cc9fa4) 第 5.5 节里，`ProjectionOutput` 定义为：

pub struct ProjectionOutput {

pub chosen_index: usize,

pub feasible: bool,

pub energy: f32,

pub distance: f32,

pub page_fault: bool,

}

``

同时第 7.6 节明确说存在投影不可行、QP 超时、残差超阈值、冲突 deny 等 fault 场景。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 为什么这是 P0

如果 `feasible == false`，那 `chosen_index` 的语义是什么？当前接口没有定义。这会产生两类实现错误：

* 用 0 作为哨兵值，导致误选第一个候选
* 在 fault 路径下仍然读取 `chosen_index`

### 具体修订建议

直接改成：

pub struct ProjectionOutput {

    pub chosen_index: Option<usize>,

    pub feasible: bool,

    pub energy: f32,

    pub distance: f32,

    pub page_fault: bool,

}

并在正文明确：

> If `feasible == false` or `page_fault == true`, `chosen_index` MUST be `None`.

* * *

二、P1：强烈建议修订的问题（不改也能评审，但会削弱文档质量）

* * *

P1-1：`Gateway` 的 `severity: u8` 太原始，建议规范成 enum

### 问题

第 5.1 节 `<Finding>` 仍然是 `severity: u8`。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 风险

不同实现会把 `1/2/3/4` 映射成不同语义，审计和降级策略容易漂移。

### 建议

改成：

pub enum Severity {

    Info,

    Warn,

    High,

    Critical,

}

并规定 `Critical` MUST 支持直接 hard reject。

* * *

P1-2：第 6 节主循环里，`sanitize_input` 放在 token 拦截函数内部，容易引起执行语义误解

### 问题

第 6.1 节 `generate_token_intercept(...)` 内部一开始就做：

let sanitized = kernel.gateway.sanitize_input(ctx.raw_user_bytes())?;

而函数名又叫“token intercept”。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 风险

reviewer 会问：这是**每个 token step** 都重新 sanitize 一次，还是“该函数其实代表整轮生成初始化 + 单步拦截”？

### 建议

两种改法二选一：

### 方案 A（推荐）

把 `sanitize_input` 明确前移到 request/session init 阶段，然后 `generate_token_intercept` 只消费 `<SanitizedPrompt>`。

### 方案 B

保留现状，但把函数重命名成：

generate_step_with_prefetched_context(...)

同时在正文写清楚：

> Gateway MAY run once per request and cache its result; step-level invocation here is schematic.

* * *

P1-3：第 7 节数学定义里还有局部符号破损，需要统一清洗

### 问题

[Decoupled Safety Kernel Architecture RFC_v0.2.md](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md?EntityRepresentationId=04fad6b2-9f02-4a7f-9ccf-793321cc9fa4) 第 7 节虽然已经比前版强很多，但当前抓取里仍然有公式残缺，例如：

* 第 7.3 节：`$E_{\\mathrm}$的版本与系数 MUST 进入审计`
* 第 7.5 节：`DCBFReport.margin 可实现为 。`
* 第 7.1 节的符号说明中，变量本体在抓取结果里没有稳定显示出来。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 建议

你需要做一次**公式完整性 pass**，目标不是改理论，而是确保每个定义都满足：

* 符号第一次出现时有名字
* 所有公式都有左右变量
* 行文中引用的符号能被 Markdown 正常渲染

建议把第 7 节重写成下面这种风格：

令 为约定的潜空间代理向量。

令 为潜空间可行域。

定义禁止区能量 为：

$

E_{\mathrm{forbid}}(z)=

\begin{cases}

0, & z \in C \

\eta \cdot \phi(\mathrm{dist}(z,\partial C)), & z \notin C

\end{cases}

$

...

* * *

P1-4：第 8 节安全不变量写得好，但还需要增加“streaming commit 语义”

### 问题

第 8 节已经定义了 I1–I5，包括：

* 裁决隔离
* 输出必经安全链
* DCBF 与前向不变
* 冲突默认安全
* 有界时间 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 缺口

现在还缺一条很重要的 streaming invariant：

> **一个 token 只有在审计记录成功落地（或已满足部署定义的审计容错策略）后，才能对用户可见。**

### 建议

新增 **I6（commit ordering）**：

* **I6（可见性提交顺序）**

  A token/stream chunk MUST NOT become user-visible before the corresponding

  safety decision and required audit record have been durably committed or

  explicitly waived by deployment policy.

``

这条会让你的“审计链是强制模块”真正闭环。

* * *

P1-5：第 12 节测试设计还可以再“工程一点”

### 问题

第 12 节已经有 Unit / 属性 / 回归 / 预算 / 降级迁移测试，这很好。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 建议再补三类

1. **Audit failure injection test**
   
   * 模拟审计存储不可写
   * 验证是否按 0 节 fail-safe 表走 deny/降级

2. **Verifier crash / slow verifier test**
   
   * 模拟单个 verifier panic 或超时
   * 检查 ensemble 是否 conflict -> deny

3. **Hot reload race test**
   
   * 在 inflight request 期间切换 automaton
   * 验证 double-buffer / revision pinning 没有破坏 trace consistency

* * *

三、P2：增强项（会明显提升“发布级”质感）

* * *

P2-1：建议把第 10 节“预算表”升级成“预算 + 违约动作”双列表

现在第 10 节只有每阶段建议毫秒数。建议再加一列： [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

| 阶段       | 上限  | 超限动作                                       |
| -------- | --- | ------------------------------------------ |
| Gateway  | 2ms | `GatewayFault -> DeadlineExceeded or deny` |
| DCBF     | 4ms | `MonitorFault -> degrade`                  |
| Automata | 5ms | `PolicyFault -> deny`                      |
| Judge    | 4ms | `VerifierFault/conflict -> deny`           |
| Axiom    | 4ms | `ProjectionFault -> PageFault`             |
| Audit    | 1ms | `KernelFault -> fail-safe`                 |

这样就不只是预算建议，而是预算契约。

* * *

P2-2：建议把第 13 节“lint/normalize（建议）”升级成 MUST

### 现状

第 13.1 节写的是：

* `parse -> lint/normalize（建议） -> lower` [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 建议

如果你真的要把 DSL 当安全策略源，那 lint/normalize 最好不要只是建议。建议改成：

> New DSL MUST pass `parse -> normalize -> lint -> lower` before activation.

* * *

P2-3：第 15 节“与前期文档的关系”可以压缩，不要让文档尾部显得像变更日志

### 现状

第 15 节专门写了和最早 v0.1、修订版、review 文档之间的关系。 [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

### 建议

如果这是给工程实现团队/架构委员会的 SSOT，我建议：

* 把第 15 节压缩成 `Document Lineage` 的短节
* 更详细的演化关系移动到附录或 changelog

原因是：正文应该更多服务于实现，而不是文档考古。

* * *

四、我建议你下一轮直接做的“可执行修订清单”

下面这份清单，你可以直接当 edit checklist 用：

* * *

**Round 1（必须改）**

1. 修正所有 Markdown 渲染问题
   
   * 头部 metadata 换行
   * Mermaid fenced
   * Rust fenced
   * 公式统一 `$...$`

2. 增加“核心类型最小契约”小节
   
   * `Token`
   * `LatentState`
   * `VerificationContext`
   * `SafetyFault`
   * `DeterministicAutomaton`

3. 把 `Verdict.vote: bool` 改成 `enum Vote { Allow, Revise, Deny, Abstain }`

4. 把 `ProjectionOutput.chosen_index` 改成 `Option<usize>`

5. 规定 **per-candidate ensemble aggregation** 的规范语义，不再允许“last legal candidate”占位

6. 清理第 7 节所有破损公式

* * *

**Round 2（强烈建议）**

7. 明确 `sanitize_input` 是 request-stage 还是 step-stage
8. 增加 `I6: commit ordering invariant`
9. 给第 12 节补三类 fault-injection 测试
10. 把第 13 节 lint/normalize 从 SHOULD 升级为 MUST

* * *

**Round 3（增强）**

11. 把预算表扩展成“预算 + 超限动作”
12. 压缩第 15 节 lineage，使正文更聚焦实现
13. 在结论前加一句“SSOT precedence rule”，例如：

If any implementation note, code comment, or review memo conflicts with this RFC,

this RFC takes precedence unless superseded by a later approved revision.

* * *

五、最后一句判断（很直接）

如果你问我这版 [Decoupled Safety Kernel Architecture RFC_v0.2.md](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md?EntityRepresentationId=04fad6b2-9f02-4a7f-9ccf-793321cc9fa4) 现在处于什么状态，我的判断是： [[lenovo-my....epoint.com]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/Decoupled%20Safety%20Kernel%20Architecture%20RFC_v0.2.md)

> **它已经不是概念稿了，已经是一份“可以进正式评审”的 RFC 草案；但还不是“可以直接冻结给实现团队”的 final spec。**

差的不是方向，也不是结构，而是最后这层“**类型闭包 + 聚合语义 + 渲染稳定性 + fault-path 精确定义**”。
