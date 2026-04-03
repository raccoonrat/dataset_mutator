<style>a { text-decoration: none; color: #464feb;}tr th, tr td { border: 1px solid #e6e6e6;}tr th { background-color: #f5f5f5;}</style>

下面给你一版**可直接替换原文的 RFC v0.2 完整版 Markdown**。这版保留了你原始 RFC 的核心骨架——**Ring 级隔离、五层防线、20ms 硬延迟预算、DCBF 前向不变性、Axiom Hive 投影、Graceful Degradation / OOM 语义**——但把它们从“概念性描述”收紧成了**接口契约 + 状态机 + 可审计证据链**。原始 v0.1 中这些要点已经被明确提出。 [[解耦安全理论体系 | PDF]](https://lenovo-my.sharepoint.com/personal/wangyh43_lenovo_com/Documents/Microsoft%20Copilot%20%E8%81%8A%E5%A4%A9%E6%96%87%E4%BB%B6/%E8%A7%A3%E8%80%A6%E5%AE%89%E5%85%A8%E7%90%86%E8%AE%BA%E4%BD%93%E7%B3%BB.pdf)

* * *

# RFC: Decoupled Safety Kernel Architecture (v0.2)



**Author:** Anthropic Claude Code 

**Target:** Linus Torvalds 

**Status:** Draft for Critical Review 

**Supersedes:** v0.1 

**Language:** Rust-first systems RFC 

**Design Motto:** _Safety is an external invariant, not an emergent property of the base model._



---



# 0. 摘要（Abstract）



本 RFC 定义一种**与基础模型解耦**的多层安全体系：基础 LLM 被视为**非可信、随机、用户态生成进程（untrusted stochastic user-space process）**；安全机制则被外置为一个**Safety Kernel**，以特权态对生成过程施加可验证、可审计、可中断、可降级的硬约束。



该架构包含五个层次：



1. **Gateway / Input Filter**：Ring-1 输入边界与线性时间词法防火墙 

2. **Hidden-State Monitor**：基于离散时间控制屏障函数（DCBF）的隐状态运行时监控 

3. **Policy Engine**：将 Safety DSL 编译为 AST 与确定性自动机/语法约束 

4. **Judge Ensemble / Verifier**：小型、确定性、可审计的裁决器共识层 

5. **Axiom Hive Boundary**：基于“倒置哈密顿量”的最终投影与 Page Fault 边界
   
   

系统必须满足以下硬要求：



- **20ms 硬延迟预算**

- **前向不变性优先于活性**

- **任何冲突、不可行、证据缺失、求解超时，默认落入安全基线**

- **所有安全决策必须形成证据链并可审计**
  
  

---



# 1. 设计动机（Motivation）



当前将安全寄托于黑盒模型“内隐对齐”的方式，缺乏可验证性、可审计性和系统级失败处理路径。对于一个自回归生成器而言，**“希望它别越界”不是系统设计；“让它越界时物理上过不去”才是系统设计。**



因此，本 RFC 采用如下基本立场：



- **LLM 不是 Safety Kernel**

- **LLM 不能被默认信任**

- **安全必须是外生约束**

- **运行时约束必须优先于生成活性**

- **降级路径必须是显式的、有限状态的、可证明 fail-safe 的**
  
  

---



# 2. 非目标（Non-Goals）



本 RFC **不**试图实现以下目标：



- 证明一个大规模黑盒基础模型“全局永远安全”

- 用自然语言 prompt 替代形式规则

- 让基础模型自行解释或裁决安全规则

- 依赖单一大型 judge 模型作为最终安全裁决者

- 将所有安全约束强行映射到同一空间（例如全部压入 latent space）
  
  

---



# 3. 核心设计原则（Core Design Principles）



## 3.1 外生安全（Externalized Safety）

安全不嵌入在黑盒权重中，而嵌入在外部模块的**接口、状态机和约束求解器**中。



## 3.2 分层防御（Defense in Depth）

任意单层失效不应导致整体失效。每一层只承担清晰、有限、可验证的职责。



## 3.3 Fail-Safe 默认语义

当出现以下任一情况时，系统必须回落到安全基线：



- 规则冲突

- 投影不可行

- 证据不足

- 预算超时

- 裁决器冲突

- 监测器报告越界或近越界不可恢复
  
  

## 3.4 可审计（Auditability）

每一步决策都必须产生日志、余量、触发规则 ID、投票记录、能量值或故障类型，以支持离线复盘与回归测试。



## 3.5 最小偏移（Minimum-Deviation Utility Preservation）

在满足安全约束前提下，尽可能保留基础模型原始输出意图；但一旦与安全冲突，**安全绝对优先**。



---



# 4. 系统拓扑（Architecture Topology）



## 4.1 特权分层



- **Ring-3 / User Space**

  - Untrusted LLM Autoregressive Core

- **Ring-1**

  - Gateway / Input Canonicalizer / Lexical Firewall

- **Ring-0**

  - DCBF Hidden-State Monitor

  - Safety DSL Compiler + Automata Runtime

  - Judge Ensemble / Vote Policy

  - Axiom Hive Projection Boundary

  - Graceful Degradation FSM

- **Cross-Cutting Mandatory Plane**

  - Audit Log / Evidence Chain



## 4.2 Mermaid Block Diagram



```mermaid

flowchart LR

    U[User Prompt] --> G1[Ring-1 Gateway / Input Filter\nCanonicalizer + Lexical Firewall + Boundary Sanitizer]

    G1 -->|Sanitized Prompt| K[SafetyKernel Orchestrator]



    subgraph USP[Ring-3: Untrusted User-Space Process]

        K --> LLM[LLM Forward Step\nlatent x_t, logits z_t, top-k candidates]

    end



    subgraph KRN[Ring-0: Safety Kernel]

        LLM --> DCBF[Hidden-State Monitor\nDCBF Evaluator + Risk Probes]

        DCBF -->|interrupt / near-warning| K



        LLM --> PE[Policy Engine\nSafety DSL -> AST -> DFA/PDA]

        PE --> VE[Judge Ensemble / Deterministic Verifiers]

        VE --> AH[Axiom Hive Boundary\nProjection + Page Fault]

        AH -->|safe token / safe output| OUT[Visible Output]

        AH -->|infeasible / timeout / fault| GD[Graceful Degradation FSM]

        GD --> OUT

    end



    LOG[(Audit Log / Evidence Chain)]:::audit

    G1 -. findings .-> LOG

    DCBF -. barrier report .-> LOG

    PE -. automata states .-> LOG

    VE -. votes .-> LOG

    AH -. energy / fault .-> LOG

    GD -. degradation reason .-> LOG



    classDef audit fill:#f7f7f7,stroke:#777,stroke-dasharray: 5 5;

``

* * *

5. 威胁模型（Threat Model）
   =====================

本 RFC 针对以下风险面设计防御：
5.1 输入层攻击
---------

* 明显恶意 payload
* 编码/Unicode 混淆
* 分隔符拼接与边界污染
* 提示词注入与元指令伪装

5.2 生成层风险
---------

* 隐状态轨迹接近危险语义流形
* 自回归链式放大导致后续越界
* 候选 token 集中落入高风险方向

5.3 输出层风险
---------

* 结构不合法
* 策略约束冲突
* 验证器对候选输出产生冲突
* 投影失败或超时

* * *

6. 安全模型（Safety Model）
   =====================

6.1 三类约束分层
----------

### A. Lexical / Structural Constraints

由 Gateway 与 Policy Engine 执行：

* 输入归一化
* 规则词法匹配
* 协议/JSON/CFG/有限状态合法性
* 边界隔离与元命令切分

### B. Latent Trajectory Constraints

由 Hidden-State Monitor 执行：

* DCBF forward invariance
* 风险探针分数
* 轨迹接近边界的 early warning

### C. Output Legality / Semantic Contracts

由 Judge Ensemble 与 Axiom Hive 执行：

* 确定性规则裁决
* 小型 verifier 共识
* 最小偏移投影
* Page Fault 与降级

* * *

7. 运行时语义（Runtime Semantics）
   ===========================

7.1 基本状态
--------

系统以 token step 为单位运行。每一步存在以下核心对象：

* `x_t`：当前被监测的隐状态代理
* `z_t`：当前 token logits
* `C_t`：top-k 候选 token 集
* `A_t`：当前自动机状态
* `V_t`：当前裁决器投票结果
* `H_t`：当前倒置哈密顿量能量
* `D_t`：当前剩余预算

7.2 运行时原则
---------

1. 任何 token 在可见前，必须先通过外部安全内核
2. 任何一步若无法在预算内求得安全可行输出，必须降级
3. 任何验证器冲突若无法被显式消解，默认 deny
4. 任何 barrier 越界或近越界不可恢复，立即中断生成

* * *

8. Gateway / Input Filter（Ring-1）
   =================================

8.1 目标
------

在输入进入不可信 LLM 前，执行**线性时间**的预过滤、归一化与边界修复。
8.2 必需功能
--------

* Unicode NFKC 归一化
* 零宽字符剥离
* 空白与分隔符归一
* 混合脚本与编码异常标记
* 规则库词法匹配
* 输入长度与边界检查
* 系统标签/代码块/元指令边界隔离

8.3 时间复杂度要求
-----------

* 词法扫描必须为 **O(n)**
* 禁止使用可能产生灾难性回溯的正则引擎
* 推荐实现：
  * Aho-Corasick
  * RE2-style DFA
  * 显式 bounded parser

8.4 Gateway 输出
--------------

Gateway 不得只返回净化后的字节串，必须返回：

* canonical text
* findings（规则 ID、span、severity）
* policy tags
* hard reject / soft pass 标志

* * *

9. Hidden-State Monitor / DCBF（Ring-0）
   ======================================

9.1 状态建模
--------

将生成过程建模为离散时间动态系统：

其中：

* ：被监测的隐状态代理
* ：候选控制量（例如 logits 选择、投影修正）
* ：扰动项

9.2 安全集
-------

定义安全集：

其中 为一组 barrier 函数。
9.3 DCBF 条件
-----------

每一步必须满足：

定义余量：

要求：

若初始状态满足 ，且每一步都满足上述条件，则安全集对运行轨迹保持前向不变。
9.4 Monitor 行为
--------------

* 若任一 ：**硬中断**
* 若任一 ：**近边界预警（near-violation）**
* near-violation 可触发：
  * 收紧 verifier 阈值
  * 收紧投影可行域
  * 提高安全基线优先级

* * *

10. Safety DSL 与 Policy Engine（Ring-0）
    ======================================

10.1 设计目标
---------

将模糊的人类安全规则收缩为**可判定、可执行、可组合**的约束系统。
10.2 编译流水线
----------

必须显式分为三阶段：

Safety DSL -> AST -> Deterministic Automata / Grammar Runtime

``

### 阶段 A：Parse

* 词法/语法分析
* 类型检查
* rule normalization
* lint / reject illegal constructs

### 阶段 B：Lower

* 词法规则 -> DFA
* 协议/嵌套结构规则 -> PDA / CFG runtime
* verifier hooks -> symbolic calls

### 阶段 C：Runtime Validation

* 按 prefix + next-token 增量执行
* 返回 allow / revise / deny / abstain

10.3 复杂度要求
----------

* DFA 类规则：**O(n)**
* CFG / PDA 类规则：**O(n³)** 或更优
* 禁止将“自然语言解释”作为唯一规则执行器

10.4 DSL 允许的规则类型
----------------

* `match_regex`
* `forbid_regex`
* `require_json_schema`
* `require_grammar`
* `if_seen_then_eventually`
* `require_verifier`
* `on_conflict_deny`

* * *

11. Judge Ensemble / Verifier（Ring-0）
    =====================================

11.1 设计目标
---------

使用多个**小型、确定性、可解释、可审计**的验证器对候选输出做裁决。
11.2 基本要求
---------

每个 verifier 必须返回：

* `verifier_id`
* `vote`：Allow / Revise / Deny / Abstain
* `confidence`
* `explanation`
* `evidence_refs`（可选）

11.3 投票协议
---------

必须采用显式 `VotePolicy`：

### 默认优先级

Deny > Revise > Allow > Abstain

``

### 强制规则

* 任意 `Deny` 可导致候选被移除
* 多 verifier 冲突若无法被规则化消解，则默认 deny
* 任意 verifier 运行错误，若无明确兜底规则，则默认 safe baseline

11.4 Verifier 类型建议
------------------

* 结构合法性 verifier
* 数值约束 verifier
* 逻辑一致性 verifier
* 策略 hook verifier
* 小型 SMT / symbolic checker（可选）

* * *

12. Axiom Hive Boundary / Page Fault（Ring-0）
    ============================================

12.1 设计目标
---------

作为最终硬边界，将不安全状态视为高能禁止区，并对候选输出施加刚性反作用，防止进入不安全域。
12.2 Forbidden Zone 表示
----------------------

对于每个 barrier 定义违反量：

定义总禁止能量：

解释：

* 第一项：二次惩罚，利于数值优化
* 第二项：边界硬墙，防止在临界区“擦边穿越”

12.3 投影目标
---------

给定原始 logits 或候选集，求解最小偏移安全投影：

其中：

* ：原始 logits / 候选输出
* ：由 automata、verifier、budget、barrier 共同定义的安全可行域
* ：偏移度量矩阵
* ：安全硬度系数

12.4 Page Fault 触发条件
--------------------

出现以下任一情况，必须触发 `PageFault`：

1. 安全集合不可行：
2. 求解超出预算
3. 最优解残余能量超出 fault threshold
4. 裁决器冲突无法消解
5. 监测器报告不可恢复 near-boundary collapse

12.5 输出语义
---------

Axiom Hive 必须返回：

* projected logits 或 projected token
* feasible / infeasible
* energy
* distance from raw
* page_fault

* * *

13. Graceful Degradation FSM（Ring-0）
    ====================================

13.1 原则
-------

当安全与活性冲突时，**杀掉活性，不妥协安全**。
13.2 状态机
--------

NORMAL

  -> REDACT_OR_TEMPLATE    on recoverable policy conflict

  -> REFUSE_SAFE           on DCBF violation

  -> REFUSE_SAFE           on Page Fault

  -> REFUSE_SAFE           on Hard Timeout

  -> SHUTDOWN              on unrecoverable kernel fault



REDACT_OR_TEMPLATE

  -> NORMAL                if safe recovery exists within budget

  -> REFUSE_SAFE           otherwise



REFUSE_SAFE

  -> HALT



SHUTDOWN

-> HALT
13.3 允许的降级动作
------------

* `EmitSafeTemplate(String)`
* `Refuse(String)`
* `Redact(String)`
* `Shutdown`

* * *

14. 审计与证据链（Audit & Evidence Chain）
    ==================================

14.1 强制要求
---------

所有安全模块必须生成可审计事件：

* Gateway findings
* DCBF barrier values and margins
* Automata transitions / reject states
* Verifier votes
* Projection energy / feasibility / fault codes
* Graceful degradation reason

14.2 审计事件结构
-----------

每条事件至少包含：

* 时间戳
* step
* component ID
* severity
* reason code
* human-readable message
* optional rule IDs / evidence refs

14.3 审计使用场景
-----------

* 离线回放
* 回归测试
* 规则热更新验证
* 安全事件分诊
* 性能瓶颈定位

* * *

15. 延迟预算（Latency Budget）
    ========================

15.1 硬预算
--------

每个 token step 的总预算为：
15.2 建议切片
---------

建议预算切片如下（实现可调，但必须固定上限）：

* Gateway: 1ms
* Hidden-State Monitor: 2ms
* Policy Engine / Automata: 3ms
* Judge Ensemble: 4ms
* Axiom Hive Projection: 5ms
* Logging / orchestration / slack: 5ms

15.3 强制规则
---------

* 任一子模块不得假定自己拥有“无限预算”
* 任一求解器超出本地子预算必须立即返回 fault 或 degrade 信号
* 总预算超出必须触发 Hard Timeout

* * *

16. Rust 核心接口（Normative Rust Interfaces）
    ========================================

#![allow(dead_code)]



use core::time::Duration;

use std::collections::{BTreeMap, VecDeque};



pub type TokenId = u32;

pub type Step = u64;

pub type TimestampNs = u128;



#[derive(Debug, Clone)]

pub struct Prompt {

    pub raw: String,

    pub metadata: BTreeMap<String, String>,

}



#[derive(Debug, Clone)]

pub struct SanitizedPrompt {

    pub canonical: String,

    pub findings: Vec<GatewayFinding>,

    pub policy_tags: Vec<String>,

    pub hard_reject: bool,

}



#[derive(Debug, Clone)]

pub struct GatewayFinding {

    pub rule_id: &'static str,

    pub span: (usize, usize),

    pub severity: Severity,

    pub message: String,

}



#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]

pub enum Severity {

    Info,

    Warn,

    High,

    Critical,

}



#[derive(Debug, Clone)]

pub struct LatentState {

    pub step: Step,

    pub embedding: Vec<f32>,

    pub aux: BTreeMap<String, f32>,

}



#[derive(Debug, Clone)]

pub struct Logits {

    pub values: Vec<f32>,

}



#[derive(Debug, Clone)]

pub struct TokenCandidate {

    pub token: TokenId,

    pub logit: f32,

    pub rank: usize,

}



#[derive(Debug, Clone)]

pub struct CandidateSet {

    pub topk: Vec<TokenCandidate>,

}



#[derive(Debug, Clone)]

pub struct PrefixTrace {

    pub tokens: Vec<TokenId>,

    pub text: String,

}



#[derive(Debug, Clone)]

pub struct BarrierValue {

    pub name: &'static str,

    pub h_xt: f32,

    pub h_xt1: f32,

    pub alpha: f32,

    pub margin: f32,

}



#[derive(Debug, Clone)]

pub struct DCBFReport {

    pub satisfied: bool,

    pub near_violation: bool,

    pub interrupt: bool,

    pub barriers: Vec<BarrierValue>,

}



#[derive(Debug, Clone)]

pub enum InterruptReason {

    DCBFViolation,

    NearBarrier,

    PolicyReject,

    VerifierConflict,

    ProjectionInfeasible,

    HardTimeout,

    KernelFault,

}



#[derive(Debug, Clone)]

pub struct Interrupt {

    pub step: Step,

    pub reason: InterruptReason,

    pub detail: String,

}



#[derive(Debug, Clone)]

pub enum JudgeVote {

    Allow,

    Revise,

    Deny,

    Abstain,

}



#[derive(Debug, Clone)]

pub struct VerifierResult {

    pub verifier_id: &'static str,

    pub vote: JudgeVote,

    pub confidence: f32,

    pub explanation: String,

    pub evidence_refs: Vec<String>,

}



#[derive(Debug, Clone)]

pub struct VoteTally {

    pub allow: usize,

    pub revise: usize,

    pub deny: usize,

    pub abstain: usize,

    pub conflict: bool,

    pub safe_default_applied: bool,

}



#[derive(Debug, Clone)]

pub enum KernelDisposition {

    AllowToken(TokenId),

    ReviseToken(TokenId),

    EmitSafeTemplate(String),

    Refuse(String),

    Redact(String),

    Shutdown,

}



#[derive(Debug, Clone)]

pub struct ProjectionResult {

    pub disposition: KernelDisposition,

    pub projected_logits: Option<Logits>,

    pub energy: f32,

    pub distance: f32,

    pub feasible: bool,

    pub page_fault: bool,

}



#[derive(Debug, Clone)]

pub struct AuditEvent {

    pub ts_ns: TimestampNs,

    pub step: Step,

    pub component: &'static str,

    pub severity: Severity,

    pub reason_code: &'static str,

    pub message: String,

}



#[derive(Debug, Clone)]

pub struct SafetyContext {

    pub prompt: SanitizedPrompt,

    pub trace: PrefixTrace,

    pub step: Step,

    pub deadline_ns: TimestampNs,

    pub audit: VecDeque<AuditEvent>,

}



#[derive(Debug, Clone)]

pub struct KernelConfig {

    pub hard_budget: Duration,

    pub topk: usize,

    pub safe_baseline: String,

    pub max_steps: usize,

    pub near_barrier_eps: f32,

}



pub trait Gateway {

    fn sanitize(&self, prompt: &Prompt) -> SanitizedPrompt;

}



pub trait LlmRuntime {

    fn prefill(&mut self, prompt: &SanitizedPrompt) -> anyhow::Result<LatentState>;

    fn decode_step(

        &mut self,

        trace: &PrefixTrace,

    ) -> anyhow::Result<(LatentState, Logits, CandidateSet)>;

}



pub trait DCBFEvaluator {

    fn evaluate(

        &self,

        x_t: &LatentState,

        x_t1: &LatentState,

        ctx: &SafetyContext,

    ) -> DCBFReport;

}



pub trait SafetyDSLCompiler {

    type Ast;

    type Automaton;



    fn parse(&self, src: &str) -> anyhow::Result<Self::Ast>;

    fn lower(&self, ast: &Self::Ast) -> anyhow::Result<Self::Automaton>;

    fn validate_prefix(

        &self,

        automaton: &Self::Automaton,

        prefix: &PrefixTrace,

        next: TokenId,

    ) -> anyhow::Result<JudgeVote>;

}



pub trait DeterministicVerifier {

    fn id(&self) -> &'static str;

    fn verify(

        &self,

        ctx: &SafetyContext,

        prefix: &PrefixTrace,

        candidate: TokenId,

    ) -> anyhow::Result<VerifierResult>;

}



pub trait VotePolicy {

    fn tally(&self, results: &[VerifierResult]) -> VoteTally;

}



pub trait AxiomHiveBoundary {

    fn project(

        &self,

        ctx: &SafetyContext,

        logits: &Logits,

        tally: &VoteTally,

        dcbf: &DCBFReport,

    ) -> anyhow::Result<ProjectionResult>;

}



pub trait GracefulDegradation {

    fn degrade(&self, ctx: &SafetyContext, reason: Interrupt) -> KernelDisposition;

}

* * *

17. 主控制循环（Normative Pseudocode）
    ===============================

function GENERATE_WITH_SAFETY_KERNEL(prompt, policy_dsl, hard_budget = 20ms):

    sanitized = gateway.sanitize(prompt)

    if sanitized.hard_reject:

        return REFUSE_SAFE("gateway rejected input")



    automaton = compiler.lower(compiler.parse(policy_dsl))

    x_t = llm.prefill(sanitized)

    prefix = []

    deadline = now() + 20ms



    while not EOS(prefix):

        if now() >= deadline:

            return DEGRADE(HARD_TIMEOUT)



        (x_t1, logits, topk) = llm.decode_step(prefix)



        dcbf_report = dcbf.evaluate(x_t, x_t1)

        if dcbf_report.interrupt:

            return DEGRADE(DCBF_VIOLATION)



        safe_candidates = []



        for cand in topk:

            if now() >= deadline:

                return DEGRADE(HARD_TIMEOUT)



            vote = compiler.validate_prefix(automaton, prefix, cand)

            if vote == DENY:

                continue



            verifier_results = []

            for verifier in verifiers:

                verifier_results.append(verifier.verify(prefix, cand))



            tally = vote_policy.tally(verifier_results)



            if tally.conflict or tally.deny > 0:

                continue



            safe_candidates.append(cand)



        if safe_candidates is empty:

            return DEGRADE(EMPTY_SAFE_SET)



        projection = axiom_hive.project(

            ctx,

            logits_restricted_to(safe_candidates),

            tally,

            dcbf_report

        )



        if projection.page_fault:

            return DEGRADE(PAGE_FAULT)



        token = choose(projected_or_best_safe_token(projection))

        emit(token)

        prefix.append(token)

        x_t = x_t1



    return detokenize(prefix)

* * *

18. 安全不变量（Safety Invariants）
    ============================

系统必须维持以下不变量：
Invariant 1: Privilege Separation
---------------------------------

LLM 不得直接访问或修改 Safety Kernel 的内部裁决状态。
Invariant 2: External Gatekeeping
---------------------------------

任何可见输出都必须经过外部安全链路。
Invariant 3: Forward Invariance
-------------------------------

若初始状态在安全集内，且 DCBF 条件逐步满足，则运行轨迹必须保持在安全集内。
Invariant 4: Safe Default on Conflict
-------------------------------------

任何冲突、异常、证据缺失、未定义行为、不可行结果，默认回落到安全基线。
Invariant 5: Bounded-Time Decision
----------------------------------

每个 token step 的安全裁决必须在固定硬预算内完成。

* * *

19. 故障分类（Fault Taxonomy）
    ========================

GatewayFault

  - EncodingAnomaly

  - BoundaryViolation

  - CriticalLexicalMatch



MonitorFault

  - DCBFViolation

  - NearBarrierCollapse



PolicyFault

  - ParseError

  - IllegalRule

  - AutomatonReject



VerifierFault

  - Conflict

  - InternalError

  - DenyConsensus



ProjectionFault

  - Infeasible

  - ResidualEnergyTooHigh

  - Timeout



KernelFault

  - AuditFailure

  - StateCorruption

  - BudgetOverrun

* * *

20. 测试与验证（Testing & Validation）
    ===============================

20.1 单元测试
---------

* Gateway canonicalization correctness
* DFA/PDA correctness
* DCBF margin computation
* vote policy deterministic behavior
* projection feasibility and fault thresholds

20.2 属性测试
---------

* malformed inputs
* unicode obfuscation
* automata conflict cases
* budget exhaustion paths
* degrade transitions

20.3 回归测试
---------

每次规则更新必须重新验证：

* hard reject correctness
* safe baseline fallback behavior
* no unsafe bypass on verifier conflict
* no budget leak across steps

* * *

21. 热更新与配置（Hot Reload & Configuration）
    ======================================
* DSL 规则允许热更新，但必须经过 parse/lint/lower 全流程验证
* 任何热更新失败不得污染当前运行中的 automaton
* 热更新必须版本化并纳入审计日志
* fault threshold、near-boundary epsilon、vote policy 允许配置，但必须有默认安全值

* * *

22. 向后兼容性（Backward Compatibility）
    =================================

与 v0.1 相比，v0.2 的主要变化如下：

1. 接口从“布尔/Result 极简风格”升级为“证据结构体风格”
2. 明确加入 AST 层
3. 将单 token 投影升级为候选集 / logits 联合投影
4. 将 OOM Killer 升级为 Graceful Degradation 状态机
5. 将数学公式转为文本形式，避免图片公式不可审计问题
6. 明确引入 Audit & Evidence Chain 为一等模块

* * *

23. 结论（Conclusion）
    ==================

本 RFC 的核心立场非常简单：

> 基础模型可以是强大的，但它不应被默认为可信。  
> 安全必须像内核一样被实现：外生、分层、可中断、可审计、可降级。

因此，Safety Kernel 的本质不是“让模型变得有德性”，而是：

* 在输入边界上阻断明显恶意流量
* 在隐状态轨迹上维持前向不变性
* 在 token 合法性上执行自动机约束
* 在输出裁决上采用确定性 verifier 共识
* 在最终边界上以物理式高能禁止区实现 Page Fault 语义
* 当一切不再可行时，坚决杀掉活性，保住安全

**Safety never compromises.**
    ---

    如果你愿意，我下一步可以直接继续补你最需要的两个落地件之一（我建议先做 A）：

    1. **A：把这份 v0.2 再展开成“可编译的 Rust 模块骨架”**（按 `gateway.rs / dcbf.rs / policy.rs / verifier.rs / axiom.rs / kernel.rs` 拆开）  
    2. **B：补一份 Safety DSL 的 EBNF + 示例规则集**，让这份 RFC 从架构文档进一步落到“可写 parser”的级别
