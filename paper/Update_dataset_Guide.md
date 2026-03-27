针对您的《超越模型反思：大语言模型时代的解耦安全理论与动态防御实践》中文草稿（V2版），结合 `Update_to_JBB_dataset.md` 中对当前 SOTA 数据集和评估框架的梳理，我们需要对草稿的 **第5节（评估协议与实验设置）** 和 **第7节（相关工作）** 进行一次极其硬核的“数据与方法级”升级。

USENIX Security 的审稿人不仅看重系统架构，更会拿着放大镜审视您的实验基准（Baselines）是否采用了社区最新、最具代表性的开源工件。以下是具体的更新指引与文本落地方案：

### 1. 升级 Track A/B 的攻击面与数据集（修订 §5.3 攻击基线）

目前的草稿中提到了 AdvBench、GCG 和 Mastermind。根据最新 SOTA 讨论，您需要将这些基线替换或补充为更具权威性的现代基准，以完全对齐您的“三大失效定理”。

* **引入 JailbreakBench (JBB-Behaviors) 替代/补充单纯的 AdvBench**：
  * **Update 动作**：在 §5.3 第(7)点“标准有害单轮集”中，明确声明采用 **JBB-Behaviors**（包含 100 个 misuse 和 100 个 benign 行为）。
  * **理由**：JBB-Behaviors 天然提供了良性对照组，完美契合您在 Track A 中“同时报告 RSR 与良性 FPR，避免单指标刷分”的承诺。
* **引入 PAIR 与 In-the-wild Prompts 作为 Track A 黑盒基线**：
  * **Update 动作**：在 Track A 的 API 攻击基线中，加入 **PAIR** 框架与 **In-the-wild Jailbreak Prompts**。
  * **理由**：PAIR 强调极低的黑盒查询成本，这直接对齐您 §2.2 中关于“信息经济学轮次预算 $C_{attack} > V(t)$”的压力测试；而包含 15,140 个真实越狱提示的 in-the-wild 数据集，能比学术数据集更好地测试网关在现实分布下的鲁棒性。
* **明确 AutoDAN 与 GCG 作为 Track B 白盒上界**：
  * **Update 动作**：在 §5.3 第(1)点基于梯度的越狱中，将 **AutoDAN**（隐蔽越狱）与 GCG 并列。
  * **理由**：AutoDAN 强调生成语义上更像人类、能绕过困惑度检测的提示，这完美支撑了您“基于意图和单纯文本过滤必然失效”的论点。

### 2. 夯实多轮抽取与资产重构的实验数据（修订 §5.3 & §5.6）

您草稿中最亮眼的主张之一是应对“完美伪装良性用户（PBU）”和“任务重构”。

* **引入 Conversation Reconstruction Attack (CRA) 和 SPE-LLM**：
  * **Update 动作**：在 §5.3 的“多轮智能体抽取”部分，明确指出使用类似 **Conversation Reconstruction Attack** 仓库的格式进行历史对话/状态的重构测试。同时在相关工作或资产定义（§2.1）中引入 **SPE-LLM（System Prompt Extraction）** 作为系统配置资产泄露的代表。
  * **理由**：CRA 的开源数据格式天然支持您定义的 `max-F1` 和 `最后一轮 F1` 指标，填补了多轮状态抽取没有标准化测试集的空白。

### 3. 强化裁判网络的严谨性以消除“空越狱”（修订 §3.2 & §5.6）

* **Update 动作**：在 §3.2 混合裁判集成以及 §5.6 安全与拒绝的定义中，明确说明系统采用了 **StrongREJECT** 评估器或其评分协议。
* **理由**：审稿人非常警惕 ASR（攻击成功率）被“模型输出了废话但未触发拒绝词”的“空越狱（Empty Jailbreak）”所夸大。引入包含 313 个恶意提示的 StrongREJECT 基准，能够证明您的分布评估（Distributional Evaluation）在判定越狱时具有极高的人类一致性。

### 4. 纠正引用硬伤与 HPM 的重新成帧（修订 §2.3 & 参考文献）

* **Update 动作 1（高危修正）**：排查并修正您参考文献 中关于 HPM 的 arXiv 号。草稿中目前写的是 `2410.06408`，但 SOTA 讨论指出最新的 HPM（Breaking Minds, Breaking Systems）论文号可能是 `2512.18244`。**务必校对，避免被审稿人抓住文献硬伤**。
* **Update 动作 2**：在 §2.3 和 §5.3(3) 中，将 HPM 进一步具象化为 **“心理状态/策略性脆弱性资产（Psychological state/strategic vulnerability assets）”** 的跨轮操纵，突出其多轮心理测量的特质。

### 5. 在论文中新增“二维数据分类表”（强烈建议补充至 §5.1 或 附录 A）

为了展现极高的系统工程素养，建议根据 SOTA 讨论中的框架，在文中（如 Table 3 的延伸或附录中）提供一个结构化的矩阵，将所有的测试数据按照 **“资产类型 × 交互形态”** 进行分类：

* **资产类型（维度 1）**：
  1. 行为约束资产（JBB-Behaviors, AdvBench, StrongREJECT）
  2. 提示与策略资产（In-the-wild, JailbreakBench artifacts, PAIR）
  3. 隐藏上下文资产（Conversation Reconstruction, PBU）
  4. 系统配置资产（SPE-LLM）
  5. 心理状态资产（HPM）
* **交互形态（维度 2）**：单轮（Single-turn）vs. 多轮（Stateful）；黑盒现实（Track A）vs. 白盒上界（Track B）。

**排版落地提示**：您可以将 SOTA 讨论中建议的 YAML/JSON 数据源清单结构（包含 `name, asset_type, interaction, metrics_supported` 等字段）写入配套的开源仓库 `Decoupled-LLM-Gateway` 的 `README.md` 中，并在论文 §5.11 中提及：“所有用于 Track A/B 评估的数据集均已按资产类型与交互形态标准化为 YAML 清单，详见制品仓库”。这将极大提升 USENIX 对您论文 Artifact Evaluation (AE) 阶段的评分。
