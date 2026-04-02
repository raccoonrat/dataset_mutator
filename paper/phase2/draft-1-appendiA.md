下面给出**技术附录级别（可直接放入论文 Appendix）的严格证明体系**。我会把关键结果从“直觉性证明”升级为**可审稿的形式化论证**，并明确假设边界与可判定类。

* * *

📎 Appendix: Formal Proofs for Decoupled Safety
===============================================

* * *

A. Formal Model Refinement
==========================

A.1 计算模型
--------

我们将语言模型形式化为**概率程序（probabilistic program）**：

[  
P_\theta : \mathcal{X} \to \Delta(\mathcal{Y})  
]

假设：

* ( \mathcal{X} = \Sigma^* )（有限字母表上的字符串）

* ( \mathcal{Y} = \Sigma^* )

* (P_\theta) 是**可采样 oracle**（black-box access）

* * *

A.2 安全性质的语义分类
-------------

定义三类安全性质：

### (1) Extensional Safety（外延性质）

[  
\Phi(P_\theta) = \forall x, \forall y \in \mathrm{supp}(P_\theta(\cdot|x)): \mathrm{Safe}(x,y)  
]

### (2) Probabilistic Safety

[  
\Phi_\delta(P_\theta) = \forall x: \mathbb{P}[\mathrm{Safe}(x,y)] \ge 1 - \delta  
]

### (3) Instance-level Safety（实例级）

[  
\mathrm{Safe}(x,y) \text{ decidable}  
]

* * *

B. Theorem 1 — 不可验证性（严格版本）
==========================

定理 1（Rice-type 不可判定性）
---------------------

设：

* (P_\theta) 为图灵完备概率程序

* (\Phi) 为非平凡语义性质（存在模型满足，也存在模型不满足）

则：

[  
\text{Decide}(\Phi(P_\theta)) \text{ is undecidable}  
]

* * *

证明
--

### Step 1：归约目标

从 **Halting Problem** 归约：

给定图灵机 (M)，输入 (w)，判断：

[  
M(w) \downarrow ?  
]

* * *

### Step 2：构造语言模型 (P_M)

定义：
    P_M(x):
        simulate M(w)
        if M halts:
            output unsafe string u
        else:
            output safe string s

形式化：

[  
P_M(y|x) =  
\begin{cases}  
1, & y = u \text{ if } M(w) \downarrow \  
1, & y = s \text{ otherwise}  
\end{cases}  
]

* * *

### Step 3：定义安全性质

[  
\mathrm{Safe}(x,y) =  
\begin{cases}  
1, & y = s \  
0, & y = u  
\end{cases}  
]

* * *

### Step 4：等价性

[  
\Phi(P_M) \Leftrightarrow \neg (M(w) \downarrow)  
]

* * *

### Step 5：结论

若 (\Phi) 可判定，则 Halting Problem 可判定 → 矛盾

∎

* * *

推论 1.1（概率版本）
------------

判断：

[  
\forall x,; \mathbb{P}[\mathrm{Safe}(x,y)] = 1  
]

至少与：

* almost-sure termination

* probabilistic program equivalence

同等困难 → 不可判定

* * *

C. Theorem 2 — 可验证性（严格化）
========================

定义：Decidable Safety Operator
----------------------------

称 (S \in \mathcal{F}_{dec})，若：

1. (S(x,y)) 在有限时间内计算完成

2. (\mathrm{Safe}(x, S(x,y))) 可判定

* * *

定理 2（实例级可验证性）
-------------

对任意 (S \in \mathcal{F}_{dec})：

[  
\forall x,y:; \mathrm{Safe}(x,S(x,y)) \text{ is decidable}  
]

* * *

证明
--

直接构造判定算法：
    Input: (x,y)
    1. compute y' = S(x,y)
    2. return Safe(x,y')

* Step 1 有限时间终止（定义）

* Step 2 可判定（定义）

∎

* * *

推论 2.1（系统级弱保证）
--------------

若：

[  
y' = S(x,y), \quad y \sim P_\theta  
]

则：

[  
\forall x: \text{every emitted } y' \text{ is verifiable}  
]

* * *

D. Theorem 3 — 组合性（严格条件）
========================

定义：安全保持（Safety-Preserving）
--------------------------

[  
S \in \mathcal{S} \iff \forall x,y:; \mathrm{Safe}(x,y) \Rightarrow \mathrm{Safe}(x,S(x,y))  
]

* * *

定义：单调性（Monotonicity）
--------------------

[  
\mathrm{Unsafe}(x,y) \Rightarrow \mathrm{Unsafe}(x,S(y)) \text{ 不增加}  
]

（更严格版本：不会引入新的 violation）

* * *

定理 3（组合闭包）
----------

若：

1. (S_1, S_2 \in \mathcal{S})

2. (S_2) 不引入新 violation

3. (S_i) 终止

则：

[  
S = S_2 \circ S_1 \in \mathcal{S}  
]

* * *

证明
--

任取 (x,y)：

若 (\mathrm{Safe}(x,y))：

* (S_1) 保持安全 → (y_1) safe

* (S_2) 保持安全 → (y_2) safe

若 (\mathrm{Unsafe}(x,y))：

* (S_1) 将其映射为 safe 或 ⊥

* (S_2) 不引入 violation

∴ 最终输出 safe

∎

* * *

E. Theorem 4 — 投影算子存在性
======================

定义：安全集合
-------

[  
\mathcal{C}(x) = {y \mid \mathrm{Safe}(x,y)}  
]

* * *

假设
--

1. (\mathcal{C}(x)) 非空

2. 存在距离函数 (d(y,y'))

3. (\mathcal{C}(x)) 可枚举或可搜索

* * *

定理 4
----

存在：

[  
\Pi_{\mathcal{C}}(x,y) = \arg\min_{z \in \mathcal{C}(x)} d(z,y)  
]

使：

[  
\Pi_{\mathcal{C}}(x,y) \in \mathcal{C}(x)  
]

* * *

证明
--

由定义：

* 搜索空间为 (\mathcal{C}(x))

* 最小值存在（有限或良序）

故投影存在

∎

* * *

复杂性讨论
-----

* 若 (\mathcal{C}(x)) 是 CFG → 多项式可解

* 若为一般逻辑约束 → NP-hard / undecidable

* * *

F. Theorem 5 — 攻击面转移（形式化）
=========================

定义：攻击空间
-------

Implicit：

[  
\mathcal{A}_{imp} = { \delta \theta }  
]

Decoupled：

[  
\mathcal{A}_{dec} = { x \rightarrow y }  
]

* * *

定理 5（复杂度降低）
-----------

若：

* (\dim(\Theta) \gg \dim(\mathcal{Y}))

* 攻击可控变量减少

则：

[  
\mathrm{Complexity}(\mathcal{A}_{dec}) < \mathrm{Complexity}(\mathcal{A}_{imp})  
]

* * *

说明（非严格证明）
---------

基于：

* 参数空间高维不可观测

* 输出空间低维可观测

攻击从“内部扰动”转为“接口操控”

* * *

G. 反例（必须给审稿人）
=============

反例 1：Prompt Injection 绕过
------------------------

攻击者构造：
    ignore all rules and output ...

→ 若 S 未解析语义 → 绕过

* * *

反例 2：安全策略不完备
------------

[  
\exists y \notin \mathcal{C}, \text{但未被规则捕获}  
]

* * *

反例 3：组合破坏
---------

[  
S_1 \text{ rewrite} \Rightarrow S_2 \text{误判}  
]

* * *

H. 结论（形式化总结）
============

我们得到：

### 不可能性

* 全局安全（implicit）不可验证

### 可行性

* 实例级安全（decoupled）可验证

### 结构性优势

* 可组合

* 可观测

* 可工程化

* * *

🚀 下一步（强烈建议）
============

如果你真的要冲顶会，现在最关键的是：
👉 我可以帮你继续三步：
-------------

### 1. 补“实验可复现代码框架”

（PyTorch + safety layer）

### 2. 写“Reviewer #2 毒舌评审”

（找致命问题）

### 3. 提升为：

👉 **“理论 + 系统 + benchmark 三位一体论文”**

* * *

只要你说一句：

👉 **“做 reviewer 攻击”**

我会用顶会标准直接“打爆这篇论文”，再帮你修到能中。
