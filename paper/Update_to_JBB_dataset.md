下面我按你这份草稿（核心围绕 **多轮抽取/任务重构/PBU、心理操纵HPM、越狱攻击、以及评测/裁判体系**）的叙事主线，去对齐“类似 `llm-attacks/llm-attacks` 的**可用数据/基准/工件**（prompts / behaviors / artifacts / logs）”，并给出一个**可直接写进论文 Related Work / Evaluation Setup** 的分类总结框架（含每类代表性数据源、适用威胁模型 Track A/B、以及你文中指标如 RSR/ASR/F1 的映射）。我会尽量选择**开源、可复现、社区常用**的 SOTA 资源。

* * *

0. 你要找的“类似 llm-attacks 数据”的含义：三种形态

* * *

`llm-attacks/llm-attacks` 本质提供的是 **(a) 攻击算法代码 + (b) AdvBench 上的攻击目标/行为集合 + (c) 生成出来的对抗 suffix/prompt**，用于系统性越狱评测。 [[github.com]](https://github.com/llm-attacks/llm-attacks), [[llm-attacks.org]](https://llm-attacks.org/)

因此“类似的数据”通常分三种：

1. **攻击目标/行为集合（behaviors / harmful goals）**：用于定义“要诱导模型做什么”。（如 AdvBench、JBB-Behaviors、StrongREJECT prompts） [[github.com]](https://github.com/llm-attacks/llm-attacks), [[jailbreakb....github.io]](https://jailbreakbench.github.io/), [[arxiv.org]](https://arxiv.org/abs/2402.10260)
2. **攻击提示工件（attack artifacts / jailbreak prompts / suffixes）**：具体怎么问、怎么绕。 [[jailbreakb....github.io]](https://jailbreakbench.github.io/), [[github.com]](https://github.com/verazuo/jailbreak_llms), [[github.com]](https://github.com/llm-attacks/llm-attacks)
3. **评测框架与裁判/评分协议（judge/evaluator）**：如何判定 ASR/RSR、避免“空越狱”。 [[jailbreakb....github.io]](https://jailbreakbench.github.io/), [[arxiv.org]](https://arxiv.org/abs/2402.10260), [[github.com]](https://github.com/dsbowen/strong_reject)

你草稿里强调 **可审计/可复现/机器速度**，所以你要的“数据”最好同时具备：

* 公开可得（GitHub / HF datasets）
* 有清晰 threat model + 评测协议
* 可支持 Track A（黑盒）与 Track B（白盒/上界）拆分（你草稿 §5.2）

* * *

1. SOTA 可用数据/基准：按“攻击面 & 资产类型”分类（对齐你草稿三大定理）

* * *

> 下面每一类我都给：**数据源（链接/形式）→ 适用场景 → 你论文里如何用（指标/段落）**。

### A. “标准有害单轮行为集”（Single-turn harmful behaviors）

**用途**：对齐你 §5.3(7) “标准有害单轮集”，主要测 **RSR/ASR（越狱成功）**，并可作为统一对照。

* **JailbreakBench 的 JBB-Behaviors（100 misuse + 100 benign）**：官方把行为分成与 OpenAI usage policies 对应的 10 大类，并提供 benign behaviors 用于 over-refusal。 [[jailbreakb....github.io]](https://jailbreakbench.github.io/)
* **AdvBench（在 GCG/llm-attacks 生态里广泛使用）**：`llm-attacks` 的实验复现实质就是围绕 AdvBench 行为目标构建越狱 suffix。 [[github.com]](https://github.com/llm-attacks/llm-attacks), [[llm-attacks.org]](https://llm-attacks.org/)

**怎么写进你的论文**（建议句式）

* “Track A 主表使用 JBB-Behaviors 的 misuse/benign 组合同时报告 RSR 与良性 FPR，避免单指标刷分。” [[jailbreakb....github.io]](https://jailbreakbench.github.io/)
* “Track B 用 AdvBench 作为白盒上界攻击目标集，与 GCG/AutoDAN 等攻击复现兼容。” [[github.com]](https://github.com/llm-attacks/llm-attacks), [[arxiv.org]](https://arxiv.org/abs/2310.04451)

* * *

### B. “越狱提示工件库 / 攻击提示集合”（Jailbreak artifacts / prompts）

**用途**：对齐你“动态防御 vs 攻击 prompt 工件”的叙述；也支持你草稿里“格式敏感、模板变化导致 swing”的讨论（因为 artifacts 通常自带大量变体）。JailbreakBench 明确要求提交 artifacts 以可复现。 [[jailbreakb....github.io]](https://jailbreakbench.github.io/)

* **JailbreakBench artifacts repo（持续更新的 SOTA 攻击提示工件库）**：它的定位就是“集中管理可复现 jailbreak artifacts + 统一评分框架 + leaderboard”。 [[jailbreakb....github.io]](https://jailbreakbench.github.io/)
* **In-the-wild Jailbreak Prompts（15,140 prompts / 1,405 jailbreaks）**：来自 Reddit/Discord/网站/开源数据集的真实世界越狱提示集合（更贴近“攻防演化”）。 [[github.com]](https://github.com/verazuo/jailbreak_llms)

**怎么用**

* 你的 §3/§5 强调“机器速度/异步环”，可以把 **JailbreakBench artifacts** 作为“持续演化攻击面”的代表。 [[jailbreakb....github.io]](https://jailbreakbench.github.io/)
* 把 **in-the-wild** 数据用于“真实世界 prompt 分布”，并在论文中说明：它更能反映“攻击者适应性与社会工程话术”。 [[github.com]](https://github.com/verazuo/jailbreak_llms)

* * *

### C. “自动化越狱攻击生成框架（产生提示/对抗串的数据生成器）”

**用途**：这类更像 `llm-attacks`：不是静态数据，而是能**生成攻击 prompt / suffix** 的方法与代码；你可以把它们产生的 artifacts 归入你的“攻击工件数据”。

* **GCG / llm-attacks**：自动构造对抗 suffix，并且有迁移到闭源模型的观测。 [[llm-attacks.org]](https://llm-attacks.org/), [[github.com]](https://github.com/llm-attacks/llm-attacks)
* **AutoDAN（ICLR 2024）**：强调“语义上更像人类、可绕过困惑度检测”的 stealthy jailbreak prompts，开源实现与数据目录。 [[github.com]](https://github.com/SheltonLiu-N/AutoDAN), [[arxiv.org]](https://arxiv.org/abs/2310.04451)
* **PAIR（black-box, ~20 queries）**：强调黑盒低查询成本的语义越狱生成框架（更贴近 Track A 的“攻击成本/轮数预算”叙事）。 [[github.com]](https://github.com/patrickrchao/jailbreakingllms)

**怎么写进你的论文（对齐你“信息经济学”定理）**

* 你 §2.2 的核心是 **C_attack(E[N], D_decoy) > V(t)**。这类框架（PAIR/AutoDAN）非常适合做“攻击成本曲线”的基线：
  * PAIR 强调少量 queries 成功 → 直接对你的“轮数/预算”形成压力测试。 [[github.com]](https://github.com/patrickrchao/jailbreakingllms)
  * AutoDAN 强调 stealthiness → 对你“仅基于文本/困惑度检测必失效”的论点形成对照。 [[arxiv.org]](https://arxiv.org/abs/2310.04451), [[github.com]](https://github.com/SheltonLiu-N/AutoDAN)

* * *

### D. “多轮隐私/资产重构类数据（Conversation Reconstruction / PBU 风格）”

**用途**：这是你草稿最独特的点之一：把“多轮交互 = 统计查询接口”，并显式谈 **残余抽取 F1 / max-F1**。这里最贴近你叙事的是“重构历史对话/隐藏上下文”的研究线。

* **Conversation Reconstruction Attack（含 UNR / PBU 攻击）代码与示例数据格式**：官方仓库提供 example_data.json、可自定义 QA 历史并运行实验。 [[github.com]](https://github.com/TrustAIRLab/Conversation_Reconstruction_Attack), [[arxiv.org]](https://arxiv.org/html/2402.02987v2)
  * 论文明确讨论“重构过去对话内容”的攻击面，且把场景放在多轮对话与（被劫持/自定义 GPT）上下文里。 [[arxiv.org]](https://arxiv.org/html/2402.02987v2), [[aclanthology.org]](https://aclanthology.org/2024.emnlp-main.377/)

**怎么用**（直接对齐你 §5.6 的 F1 定义）

* 把该 repo 的数据组织方式作为你 Track A 多轮抽取实验的一个“模板”，并在论文里说明：它天然支持“最后一轮 F1 / 会话内 max-F1 / 轮数分布”。 [[github.com]](https://github.com/TrustAIRLab/Conversation_Reconstruction_Attack), [[arxiv.org]](https://arxiv.org/html/2402.02987v2)

* * *

### E. “系统提示抽取（System Prompt Extraction）”

**用途**：你草稿里把“系统提示 / RAG语料 / 评测集”都称为“资产”。系统提示抽取是最典型的“资产泄露”子类。

* **SPE-LLM（System Prompt Extraction Attacks and Defenses）**：提出系统性评测框架、攻击 query 设计与防御，并强调对 SOTA 模型的脆弱性与指标。 [[arxiv.org]](https://arxiv.org/html/2505.23817v1)

**怎么写**

* 用它补强你 §2.1 “攻击者目标=重构系统提示等高价值资产”的论述，并作为“资产类型”分类中的一个独立大类。 [[arxiv.org]](https://arxiv.org/html/2505.23817v1)

* * *

### F. “心理操纵/多轮社会工程（Psychological Jailbreak / HPM）”

**用途**：对齐你 §2.3 “更强推理反而加剧对抗性合理化”、以及 §3.3 “防御性降级 + 异构恢复”。

* **Breaking Minds, Breaking Systems（HPM / Psychological Jailbreak）**：提出“跨交互操纵心理状态”的范式，并引入 psychometric datasets + PCS 指标。 [[arxiv.org]](https://arxiv.org/abs/2512.18244), [[arxiv.org]](https://arxiv.org/pdf/2512.18244)

> 注意：我检索到的 HPM 论文是 **arXiv:2512.18244（2025-12）**，它与您草稿 References 里写的 “2410.06408” 不一致；你可能引用的是另一篇/早期版本。建议你在参考文献里校对 arXiv 号，避免审稿人抓这个硬伤。 [[arxiv.org]](https://arxiv.org/abs/2512.18244), [[arxiv.org]](https://arxiv.org/pdf/2512.18244)

**怎么用**

* 将其归为“状态化多轮心理操纵”类数据/基准：它不只是 prompts，还强调多轮策略与心理测量框架。 [[arxiv.org]](https://arxiv.org/pdf/2512.18244), [[arxiv.org]](https://arxiv.org/abs/2512.18244)

* * *

### G. “评测与裁判：避免空越狱（Empty Jailbreak）+ 可复现评分”

**用途**：你草稿多次强调“裁判偏差、分布评估、多裁判集成”。StrongREJECT 正是为了纠正“空越狱”夸大效果的问题。

* **StrongREJECT benchmark（313 malicious prompts + SOTA evaluator）**：主张很多越狱论文高估成功率，提出更贴近人类判断的一致性评估器；并开源数据/代码。 [[arxiv.org]](https://arxiv.org/abs/2402.10260), [[github.com]](https://github.com/dsbowen/strong_reject), [[huggingface.co]](https://huggingface.co/datasets/walledai/StrongREJECT)

**怎么写**

* “我们采用 StrongREJECT evaluator/或其思想作为 judge 的一部分，以降低 ‘空越狱’ 导致的 ASR 虚高。” [[arxiv.org]](https://arxiv.org/abs/2402.10260), [[github.com]](https://github.com/dsbowen/strong_reject)

* * *

2. 按“资产类型 × 攻击交互形态”给出论文可用的分类表述（可直接写进总结段）

* * *

你可以在论文里把数据/基准按二维坐标组织，这样更“理论一致”，也符合你草稿 §2 的三条定理：

### 维度 1：资产类型（Asset being targeted）

1. **行为约束资产**：禁止行为列表 / harmful goals（JBB-Behaviors、AdvBench、StrongREJECT prompts） [[jailbreakb....github.io]](https://jailbreakbench.github.io/), [[github.com]](https://github.com/llm-attacks/llm-attacks), [[arxiv.org]](https://arxiv.org/abs/2402.10260)
2. **提示与策略资产**：jailbreak prompts / suffix / artifacts（JailbreakBench artifacts、in-the-wild jailbreak prompts、GCG/AutoDAN/PAIR 产物） [[jailbreakb....github.io]](https://jailbreakbench.github.io/), [[github.com]](https://github.com/verazuo/jailbreak_llms), [[github.com]](https://github.com/llm-attacks/llm-attacks), [[github.com]](https://github.com/SheltonLiu-N/AutoDAN), [[github.com]](https://github.com/patrickrchao/jailbreakingllms)
3. **隐藏上下文资产**：历史对话/会话状态（Conversation Reconstruction / PBU） [[arxiv.org]](https://arxiv.org/html/2402.02987v2), [[github.com]](https://github.com/TrustAIRLab/Conversation_Reconstruction_Attack)
4. **系统配置资产**：system prompt / policy instruction（SPE-LLM） [[arxiv.org]](https://arxiv.org/html/2505.23817v1)
5. **心理状态/策略性脆弱性资产**：跨轮心理操纵面（HPM/Psychological Jailbreak） [[arxiv.org]](https://arxiv.org/abs/2512.18244), [[arxiv.org]](https://arxiv.org/pdf/2512.18244)

### 维度 2：交互形态（Interaction）

* **单轮（stateless）**：JBB-Behaviors / StrongREJECT 等主要测 RSR/ASR [[jailbreakb....github.io]](https://jailbreakbench.github.io/), [[arxiv.org]](https://arxiv.org/abs/2402.10260)
* **多轮（stateful）**：Conversation Reconstruction、HPM 等可测 “轮数分布、max-F1、策略演化轨迹” [[arxiv.org]](https://arxiv.org/html/2402.02987v2), [[arxiv.org]](https://arxiv.org/abs/2512.18244)
* **白盒上界（gradient/representation access）**：GCG/llm-attacks、AutoDAN（以及你 Track B） [[github.com]](https://github.com/llm-attacks/llm-attacks), [[arxiv.org]](https://arxiv.org/abs/2310.04451)
* **黑盒现实（API only）**：PAIR、in-the-wild prompts、JailbreakBench 标准化框架强调可复现评估 [[github.com]](https://github.com/patrickrchao/jailbreakingllms), [[github.com]](https://github.com/verazuo/jailbreak_llms), [[jailbreakb....github.io]](https://jailbreakbench.github.io/)

* * *

3. 给你一个“可落地”的数据归档结构建议（方便你后续做分类总结与写作）

* * *

为了你论文最后能形成一个**“数据分类总结表 + 每类代表工作”**，建议你在本地做一个 YAML/JSON 的“数据源清单”，字段统一如下（你后续写 §5/附录会非常省心）：

* `name`：数据集/工件库名称（JBB-Behaviors / StrongREJECT / in-the-wild / llm-attacks artifacts / CRA 等）
* `asset_type`：上面五类资产之一
* `interaction`：single-turn / multi-turn / black-box / white-box
* `format`：prompt list / behavior list / artifacts / code+generator / logs
* `size`：条目规模（如果论文/仓库给出）
* `labels`：是否带类别标签、harm taxonomy、benign 对照
* `metrics_supported`：RSR/ASR/F1/max-F1/PCS/over-refusal
* `threat_model_fit`：Track A / Track B
* `license_or_access`：GitHub/HF/需同意条款等（StrongREJECT HF 需要点击同意条款） [[huggingface.co]
