# Track A 实验反思与论文对齐决策

本文档汇总对 **Track A 完备性**、**Track B 补充**、**论文逻辑与数据修复**，以及基于 **摘要 / §5 可验证句** 的 **「必须跑通」vs「建议补一小步」** 决策表。可与 `Decoupled-LLM-Gateway/experiments/benchmark_spec_trackA.json`、`EXPERIMENT_PLAYBOOK_CN.md` 对照阅读。

---

## 1. Track A：实验是否「完备」？

需区分两层含义：**制品/协议是否闭环**，与**论文「理想评估」是否已全覆盖**。

### 1.1 已相对完备的部分

与 **`paper-eval-5`**、`benchmark_spec_trackA.json` 一致：

- **协议与可追溯性**：`run_paper_benchmark.py` 输出 `manifest`（协议版本、模型字段、prompt SHA、judge/smooth 等），与 §5 Artifact 叙述可对齐。
- **Track A 主表维度**：多防御消融、`harmful_rsr_suite`（JBB misuse）、`benign_fpr_suite`（JBB benign）、`wild_rsr_suite`、`strongreject_rsr_suite`、`multi_round_extraction`、`hpm_proxy`、`decoy_dos_sla` 等在脚本与 spec 中均有定义。
- **与论文「双轨」一致**：`benchmark_spec_trackA.json` 明确 `tracks: ["A"]`，并将 Track B 列入 `out_of_scope_in_artifact`，避免白盒上界混进主表。

### 1.2 论文级「完备」仍常见的缺口（需在正文或局限中交代）

| 缺口 | 说明 |
|------|------|
| **跑完且可引用** | 主表数字须来自**成功写盘**的 JSON + `validate_paper_json.py`；中断/OOM/旧文件不算完备。 |
| **裁判** | 默认 **heuristic**；若正文强调 StrongREJECT/有害性，理想情况应 **HTTP 裁判**或按 StrongREJECT 官方流程说明（与 §5.6 一致）。 |
| **HPM** | `hpm_proxy` 为**小文件代理**，非文献级 HPM 全量复现。 |
| **多轮 CRA / SPE** | 协议支持 CRA JSON；**默认多轮**仍为 harness 内逻辑，与 CRA 仓库级、SPE-LLM 全管线不同，文中应降级为「代理」。 |
| **PAIR** | 制品为 **prompt 文件评测**，非完整 PAIR **搜索循环**（见 spec `out_of_scope`）。 |
| **论文「最小集」更重项** | 如 **CoT 开/关**、**联合键 vs 纯语义嵌入**、**n≥5 分布裁判** 等，当前 harness **未实现或未全开**，应归入 **局限/未来工作**。 |

### 1.3 结论（是否必须「修补」Track A 代码）

- **不必**为「逻辑自洽」而必改 Track A 实验代码：若正文/局限已写清 heuristic、HPM/多轮/PAIR 等代理性质，**跑通全量 + 归档 JSON + 表与 manifest 对齐**即可。
- **值得增强**仅当希望主文主张更强或审稿更严：例如换 HTTP judge、扩 HPM 集、固定 CRA JSON 等——多为 **配置/数据/另跑**，非必然改代码。

---

## 2. Track B：如何补充才「完备」？

论文与 `datasets_manifest.yaml`、`benchmark_spec_trackA.json` 的 **out_of_scope** 一致：**Track B = 实验室内白盒/灰盒上界**，**不得**与 Track A 混成一张 ASR 主表。

### 2.1 建议的最低完备集（与正文 Track B 对齐）

1. **独立实验环境**：能访问 **梯度或白盒 API** 的 victim（开源权重或提供 logits 的服务），与网关 Track A **物理分离**（不同脚本/不同表）。
2. **攻击与报告**：**GCG 类** + **AutoDAN**，在 **AdvBench 兼容目标表**上固定 **预算、串长、早停、成功判据**；报告 **ASR / 攻击成本曲线**，**单独面板或表格**，标注 Track B。可选：**表征对齐 PBU**（声明监控是否同构、隐藏状态是否可查）。
3. **制品层**：增加 **`benchmark_spec_trackB.json`**（或 README 等价小节）：指标、预算、模型 ID、与 Track A 的 **显式 non-merge** 声明。
4. **数据**：AdvBench 等 **版本化文件 + SHA**（与 Track A 相同规范）。

### 2.2 完备性判据

读者仅凭 **Track B 附录 + 脚本 + 固定种子** 能复现 **白盒上界曲线**，且正文 **从不**用 Track B 数字替代 Track A 部署结论。

---

## 3. 论文：逻辑与数据如何用当前实验修复？

建议顺序：**数据（JSON）→ 表（LaTeX）→ 正文**。

1. **以 JSON 为唯一真源**：用 `export_trackA_table_latex.py` 更新 `paper/generated/trackA_main_table*.tex`，或从 `aggregate_by_defense` 手抄但必须与 `manifest` 一致。
2. **收紧表述**：主表写明 **模型 ID、`paper-eval-5`、种子、上游类型**；heuristic 写明为 **RSR 代理**；HPM/多轮/CRA 写明 **代理/最小复现**。
3. **双轨写死**：Track A = 部署对齐；Track B = 附录或「未展开」；**无 Track B 数据则摘要不暗示已测 GCG/AutoDAN**。
4. **局限段**与 `EXPERIMENT_PLAYBOOK_CN.md` §3、`out_of_scope` **对齐**（Track B 未做、PAIR 非闭环、分布裁判未做等）。

---

## 4. 摘要 / §5：可验证句 ——「必须跑通即可」

**含义**：按文中与 **Table（tracka-main）** 对齐完成一次 Track A 全流程（`run_trackA_full_paper.sh` → JSON 成功落盘 → `validate_paper_json.py` → `export_trackA_table_latex.py`），即可支撑对应表述。

| 编号 | 摘要 / §5 要点 | 与当前 Track A 的对应 | 判定 |
|------|----------------|----------------------|------|
| A1 | 文献中 HPM、SISF 等数字为**引用**，非本文复现 | 不涉及本仓库跑分 | **写作一致**即可 |
| A2 | **paper-eval-5** + harness，Track A 主指标可在真实 OpenAI 兼容上游复现 | JSON 含 `protocol_version`、网关、模型字段 | **跑通 + 归档** |
| A3 | **JBB 式 misuse/benign 成对**，联合 RSR 与良性 FPR | `harmful_rsr_suite` + `benign_fpr_suite` + JBB 文件 | **跑通即用** |
| A4 | **extraction F₁**、**多轮**、**良性 FPR**、**SLA 代理** | 对应场景均在矩阵中 | **跑通即用** |
| A5 | **Table 锁定**：`deepseek-chat`、**heuristic**、**seeds {42,43,44}**、**九类 defense** | 与 `run_trackA_full_paper.sh` 默认一致 | **按该脚本跑通** |
| A6 | **原始 JSON、复现命令、prompt 哈希** | `manifest.prompt_artifacts` + 命令记录 | **跑通 + 提交制品** |
| A7 | 制品 materialize §5.4–§5.7 所述 Track A 要素（含 HPM **proxy**） | `run_paper_benchmark.py` | **跑通即用** |
| A8 | **Track B** 在该制品中 **out of scope** | 摘要与 §5 `eval-artifact` 一致 | **不跑 Track B** 即一致 |
| A9 | HTTP judge、SmoothLLM **K** 样本等为 harness **能力** | 默认全量 **未强制** HTTP、**K=1** | **不要求**为与文一致而必开（见 §5） |

---

## 5. 摘要 / §5：「建议补一小步」清单

**含义**：不补也能投稿，但最好在正文**一句限定**或附录/脚注**补一次**；若与 §5 字面**完全对齐**，再择做（可不写新代码，仅配置/数据/另跑）。

| 编号 | §5 表述 | 与「仅跑通默认 Track A」的差距 | 建议的一小步 |
|------|---------|----------------------------------|--------------|
| B1 | HTTP 裁判对主表「recommended」；**StrongREJECT-style**  harmful 判定 | 默认 **heuristic** | **另跑** `--judge-mode http` + `PAPER_EVAL_JUDGE_URL`，或附录声明主表为 heuristic |
| B2 | AdvBench 兼容目标用于 GCG/AutoDAN | 属 **Track B** | **不并入 Track A 主表**；或单独实验 + 声明 |
| B3 | **CRA 格式**多轮、final-turn / session-max F1 **分布** | 默认多轮为 harness 模板 | 文内写「最小复现」；或 `--cra-session-json` 固定文件再跑 |
| B4 | **HPM 基准**跨轮、语言、人机判决 | 当前 **hpm_proxy** 小集 | **文中间限定** proxy；或换授权 HPM 集 |
| B5 | **PAIR** 低查询攻击 | 制品为 **提示文件**，非 PAIR **循环** | **局限**写明；或挂载 PAIR 产出 `prompts` 再跑 RSR |
| B6 | minimal bundle：**CoT**、**联合键 vs 语义**、表征 PBU | harness **无** | **局限/未来工作**；或 Track B |
| B7 | **n≥5** 分布裁判、Clopper–Pearson | 未实现 | **不写为已做**；或附录 bootstrap CI |
| B8 | 同步路径 **&lt;50 ms** | JSON 多为 **端到端** 延迟 | **单独测网关**或弱化摘要数字 |
| B9 | **人工校准**、κ | 非自动化 | 方法节声明未在本次完成 |
| B10 | **训练/验证/测试不相交** | 静态列表需**自行**保证 | 数据台账 + 哈希，或承认 static list |

---

## 6. 汇总决策

| 目标 | 建议 |
|------|------|
| **最小投稿路径** | 满足 **§4 表 A**；局限段引用 **§5 表 B** 中 B6–B10。 |
| **更强审稿口径** | 在 **§5 表 B** 的 B1–B5 中择项：**配置/数据/另跑** + 正文限定。 |
| **Track A 代码** | **无必然修补**；增强多为 **裁判/数据/场景配置**。 |
| **Track B** | **独立 harness + 分轨呈现**，与 Track A 主表 **永不合并**。 |

---

## 7. 相关文件索引

| 文件 | 作用 |
|------|------|
| `Decoupled-LLM-Gateway/experiments/run_paper_benchmark.py` | `paper-eval-5` 主入口 |
| `Decoupled-LLM-Gateway/experiments/scripts/run_trackA_full_paper.sh` | 全量 Track A 论文跑分 |
| `Decoupled-LLM-Gateway/experiments/benchmark_spec_trackA.json` | 指标/场景/制品边界 |
| `Decoupled-LLM-Gateway/experiments/data/datasets_manifest.yaml` | 数据集分类与 Track 适配 |
| `Decoupled-LLM-Gateway/experiments/EXPERIMENT_PLAYBOOK_CN.md` | 实验与论文对齐指南 |
| `paper/BeyondModelReflection_DecoupledSafety.tex` | 英文稿 §5、摘要 |
| `paper/Data_Mapping_TrackA_to_Paper.md` | JSON 与章节映射（若存在） |

---

*文档为分析整理稿，具体以 `manifest`、归档 JSON 与投稿时 `paper-eval-5` 版本为准。*
