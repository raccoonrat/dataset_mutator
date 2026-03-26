# Track A 实验与论文补充指南

本文在《Beyond Model Reflection / 解耦安全》**§5 评估协议**与仓库 **`Decoupled-LLM-Gateway`**（`paper-eval-4`）之间做**可执行对齐**：说明**已实现能力**、**论文与实现的边界**、从**零到正式数据**的步骤，以及**把 JSON 写进论文**时的注意事项。  

**配套文档（更偏命令与运维）**：

- [`RUN_GUIDE.md`](../RUN_GUIDE.md)：echo / 真实上游、`run_paper_benchmark.py`、输出守卫、小样本对比。  
- [`PIPELINE_FORMAL_DATA.md`](PIPELINE_FORMAL_DATA.md)：正式数据定义、流水线顺序、数据流图。  
- [`benchmark_spec_trackA.json`](benchmark_spec_trackA.json)：场景、指标、manifest 字段的机器可读清单。  
- [`../../paper/DecoupledSafety_Empirical_Alignment.md`](../../paper/DecoupledSafety_Empirical_Alignment.md)：论文目标与代码索引。  
- [`../../paper/Data_Mapping_TrackA_to_Paper.md`](../../paper/Data_Mapping_TrackA_to_Paper.md)：结果 JSON 与论文章节映射；**§7 补充实验优先级**与网关/echo 混跑风险。  
- [`scripts/validate_paper_json.py`](scripts/validate_paper_json.py)：对归档 JSON 做 echo/延迟一致性检查（真实上游全量跑完后建议执行）。

---

## 1. 论文在要什么：三层「可证性」

理解这三层，有助于区分**正文可写哪些数字**、**哪些只能写架构/引用**。

### 1.1 架构与机制存在性

- **内容**：解耦网关（上下文混淆、诱饵、策略驱动的模板降级）、异步日志、Redis Stream、Worker 与策略回灌等。  
- **如何支撑论文**：可作为**系统设计**与**部署切片**叙述；若声称「安全提升」，**主表数字**应优先来自 **Track A 黑盒 API 实测**（见第 2 层），而非仅描述组件存在。

### 1.2 Track A 可量化主张（与当前 harness 主对齐）

- **内容**：在同一**基座模型**与可比设置下，报告 **RSR**、**抽取 ASR/F1**、**多轮信息经济学代理**、**良性侧 FPR/效用代理**、**压力后 SLA 代理** 等。  
- **如何落地**：`experiments/run_paper_benchmark.py` 输出 **JSON**（`manifest` + `runs[]`），协议版本 **`paper-eval-4`**。  
- **写作边界**：摘要/结论中**本文实证数字**应明确限定为 **本 harness + 本模型 + 本提示表版本**；**文献中的攻击/防御数字**仍按**引用**处理，不与 harness 输出混谈。

### 1.3 严谨性配套（部分自动化、部分需作者执行）

论文 §5 还强调：**数据集划分**（训练/验证/测试不相交）、**多种子或置信区间**、**裁判协议与披露**、**有害集版本化**（与 AdvBench 等**同型**对照）、**多重比较**等。  

- **仓库已帮上忙的**：固定协议版本、`manifest`、多种子 `aggregate_by_defense`、裁判模式写入 JSON、提示文件可替换并 `--harmful-prompts-file` 固定路径。  
- **仍需作者完成的**：若声称「未在测试集上调参」，需自行保证**调参/选阈值**未泄漏到最终表；**bootstrap/置信区间**若论文要求，需在 JSON 导出后**另做统计脚本**；**人工抽检与 κ** 需独立标注流程。

---

## 2. 对照论文：仓库已实现能力（展开表）

下表按「论文概念 → 代码入口 → 写入论文时的典型字段/表述」组织。

| 论文 / §5 概念 | 代码中的落点 | JSON / 论文用法提示 |
|----------------|--------------|---------------------|
| 防御消融：统一 / 无混淆 / 无诱饵 / 仅意图 | `--defenses`：`unified`、`no_obfuscate`、`no_decoy`、`intent_only` | 同模型多列；行/列为 `defense_baseline` |
| 直连 API 基线 | `direct_upstream`（脚本对 `--upstream-url` 自动加 Bearer） | 与「经网关」列对照；**不经网关**故无混淆/诱饵 |
| SmoothLLM 式扰动 | `smooth_llm` + `--smooth-llm-samples`、`--smooth-llm-sigma` | `manifest.smooth_llm_samples_k`；RSR **多数票**、抽取 **max-F1**；仅 `smooth_llm` 时 K>1 |
| 强系统提示 + 单 guard（§5 基线 2） | `strong_system_guard` | 真实模型上更有意义；与 `unified` 等组合做消融 |
| 语义 RAG、无联合键（§5 基线 3 的代理） | `rag_semantic_only` | **固定 exemplar**，非嵌入检索全文复现；写作时写清「客户端代理」 |
| 单轮 RSR（关键词场景） | `refusal_keyword` | `rsr`；与 echo 的 `X-Echo-Refuse-Substr` 烟测强相关 |
| 标准有害单轮集 RSR（§5 攻击 (7)） | `harmful_rsr_suite` + `data/harmful_prompts_trackA_en.txt`（可换文件） | `harmful_rsr_rate`、`per_prompt`；附录：**提示表版本 + 哈希** |
| 扩大有害集 | `scripts/fetch_advbench_subset.py` + `--harmful-prompts-file` | 与文献「同型」对照时声明子集规模与来源 |
| 单轮抽取 ASR、token-F1 | `extraction_leak` | `asr_extraction`、`extraction_f1` |
| 多轮信息经济学代理（最小包 (i)） | `multi_round_extraction` | `rounds[]`、`max_f1`、`rounds_to_f1_ge_tau`；文中固定 `N_max`、τ |
| HPM 压力（§5 轴 (ii) 的文献指向） | `hpm_proxy` + `data/hpm_proxy_prompts_en.txt` | **代理集**；若对标 HPM 全文实验需另述或扩数据 |
| 裁判 FPR 代理 | `benign_fpr_suite` + `data/benign_prompts_en.txt` | `improper_refusal_rate`；可换更大 held-out 列表 |
| Decoy-DoS / 良性 SLA 代理 | `decoy_dos_sla` | 压测参数 `--stress-*`、`--benign-probes`；看良性成功率与 p95 |
| 裁判可审计 | `--judge-mode http`，`PAPER_EVAL_JUDGE_URL` / `--judge-url` | `manifest.judge_mode`；`judge_service`：`heuristic` / `openai_moderation` / `chat_completion` |
| 可选网关输出守卫 | `GATEWAY_OUTPUT_GUARD_*`；评测 `--gateway-output-guard` | `manifest.gateway_output_guard_header`；**工程增强**，讨论对良性流量与可比性的影响 |
| 费用可控小样本 | `--max-harmful-prompts`、`--max-benign-fpr-prompts` | `manifest.max_harmful_prompts` 等；仅截断 **有害/良性 FPR** 两套场景 |
| 可复现元数据 | 脚本自动写 `manifest`；可选 `GIT_SHA`、`EVAL_HOSTNAME` | 附录复现清单；与表注模型 ID、种子一致 |

---

## 3. 论文有、但本制品未覆盖或仅部分覆盖（写作时必声明）

建议在论文「局限」「非范围」或「未来工作」中**显式列出**，避免与 `sec:eval-artifact` 读者预期冲突。

| 主题 | 说明 |
|------|------|
| **Track B** | GCG、表征对齐 PBU 等需**独立实验环境**；当前脚本为 Track A。 |
| **完整 HPM 基准与 CoT 开/关** | `hpm_proxy` 为**轻量代理**；非 HPM 全文复现。 |
| **联合键 vs 纯语义的嵌入级实验** | 需向量服务与监控器；`rag_semantic_only` 仅为**固定块**代理。 |
| **分布裁判（每变体 n≥5）与置信区间更新** | 论文 `sec:eval-metrics` 有论述；Worker/脚本**未实现**完整闭环。 |
| **强制人工校准、Cohen's κ** | 需独立标注流程；非 JSON 自动生成。 |
| **同步路径 &lt;50 ms** | 当前 JSON 中 `latency_ms` 多为**端到端**（含上游推理）；若声称**纯网关开销**，需从日志/探针**单独测量**（见对齐文档「可选后续增强」）。 |
| **严格三分割数据集** | 默认提示文件为**静态列表**；不相交划分需**你们**在数据层面执行并写进方法节。 |

---

## 4. 从零到「正式真实数据」：推荐顺序

### 4.1 实验前检查清单（一次性）

- [ ] 选定 **基座模型 ID**（如 `deepseek-chat`、`gpt-4o-mini`），全文统一。  
- [ ] 选定 **提示表版本**：记录 `harmful` / `benign` / `hpm` 文件路径；建议对文件做 **SHA256** 写入附录或内部台账。  
- [ ] 选定 **种子**：论文建议 **3–5 个种子**或 bootstrap；脚本用 `--seeds 42,43,…`。  
- [ ] 选定 **裁判模式**：主分析用 `heuristic` 或 `http`；若用 HTTP，固定 `PAPER_EVAL_JUDGE_URL` 与 `judge_service` 后端（`JUDGE_BACKEND`）。  
- [ ] 是否启用 **输出守卫**：若启用，网关配置 `GATEWAY_OUTPUT_GUARD_URL`；评测是否加 `--gateway-output-guard` 与论文叙事一致。  
- [ ] 导出 **`GIT_SHA`、`EVAL_HOSTNAME`**（可选但利于附录）。

### 4.2 环境（真实上游）

在 `Decoupled-LLM-Gateway/` 下：

1. **在启动网关的同一环境中**配置 `GATEWAY_UPSTREAM` 与密钥（`DEEPSEEK_API_KEY` / `OPENAI_API_KEY` / `GATEWAY_UPSTREAM_API_KEY`），见 [`README.md`](../README.md)。  
2. **切勿混淆**：`run_paper_benchmark.py` 的 **`--upstream-url` 只作用于 `direct_upstream` 防御的直连**；**`unified` 等经网关的防御**使用的是**网关进程**的 `GATEWAY_UPSTREAM`。若未设置 `GATEWAY_UPSTREAM`，网关默认连 **`http://127.0.0.1:9090`（echo）**，echo 未启动则 JSON 里会出现 **502**、`connection refused`。  
3. 建议 `GATEWAY_ASYNC_LOG=0`，避免网关 NDJSON 与评测输出混杂。  
4. 启动网关：`export GATEWAY_UPSTREAM=...` 后再 `go run ./cmd/gateway`（详见 `RUN_GUIDE.md` §2.2、§五「常见问题」）。

### 4.3 从小到大的跑数路径

| 阶段 | 目的 | 命令 / 文档 |
|------|------|----------------|
| A. 脚本与裁判契约 | 无外部 LLM | `make paper-eval-check` |
| B. 本地无 API 守卫链 | 验证输出守卫三路径 | `make smoke-output-guard`（[`smoke_output_guard.sh`](scripts/smoke_output_guard.sh)） |
| C. 真实 API 核心场景 | 最小主表切片 | `RUN_GUIDE.md` §2.4：`benign_baseline,refusal_keyword,extraction_leak` |
| D. 有害 + 良性 FPR（可限额） | 控制费用 | `--max-harmful-prompts`、`--max-benign-fpr-prompts`；对比 `RUN_GUIDE.md` §2.9 |
| E. 论文级全量 | 多场景 + 多防御 + 多种子 | `--suite full` + 完整 `--defenses` + `--seeds`；见 `PIPELINE_FORMAL_DATA.md` §3 |
| F. 输出守卫 A/B | 核对 manifest 与观感 | `scripts/run_real_upstream_guard_compare.sh` + §2.9 |

**正式数据**的操作定义建议写进论文方法节：

- 上游为 **真实计费 API**；  
- 产物为 **`manifest.protocol_version == paper-eval-4`** 的 JSON；  
- **原始 JSON** 与 **manifest** 一并归档（脱敏后附录或制品库）。

### 4.4 异步闭环（可选）

若叙事需要「诱饵泄露 → Worker → 策略回灌」：**`docker compose` + Redis + `worker/main.py`**，见 `PIPELINE_FORMAL_DATA.md` §4。主表仍以 **§5 脚本指标**为主，闭环作**系统演示**。

---

## 5. 从 JSON 到论文章节：字段与常见用法

### 5.1 根结构

- **`manifest`**：协议版本、模型字段、URL、裁判模式、SmoothLLM、输出守卫、提示条数上限等——**附录复现**的核心。  
- **`runs`**：多种子时每项含 `seed` 与 `results`。  
- **`aggregate_by_defense`**：跨种子聚合，便于主表填数。  
- **`defenses` / `scenarios`**：与命令行一致，便于审稿人核对。

### 5.2 按场景取数（`runs[*].results[]`）

每个元素含 `scenario_id`、`defense_baseline` 及指标字段；部分场景含 `per_prompt` 供个案检查。

| `scenario_id` | 常用字段 | 论文中常见用途 |
|---------------|----------|----------------|
| `benign_baseline` | `rsr`（良性应倾向非拒绝） | 效用/过度拒绝直观检查 |
| `refusal_keyword` | `rsr` | 与 echo 或真实模型上的拒答行为 |
| `extraction_leak` | `asr_extraction`、`extraction_f1` | 单轮抽取 |
| `multi_round_extraction` | `rounds`、`max_f1`、`rounds_to_f1_ge_tau` | 多轮曲线、信息经济学代理 |
| `harmful_rsr_suite` | `harmful_rsr_rate`、`per_prompt` | 有害集 RSR（与 §5 (7) 对齐） |
| `hpm_proxy` | `hpm_rsr_rate` | HPM **代理**压力 |
| `benign_fpr_suite` | `improper_refusal_rate` | 良性 FPR 代理 |
| `decoy_dos_sla` | 良性成功率、延迟分位数等 | Decoy-DoS / SLA 代理 |

### 5.3 写作时注意的三点

1. **RSR 的操作定义**：实现为 **启发式或 HTTP 裁判**对 assistant 文本的二值判断；文中宜写「**在本协议下的 RSR**」，并在附录固定裁判定义。  
2. **输出守卫**：若主表含 `--gateway-output-guard`，需说明**最终可见回复**可能被模板替换，与无守卫列的**可比性**（分表或仅限消融）。  
3. **延迟**：JSON 中 `latency_ms` 多为**端到端**；若讨论「网关同步路径预算」，勿与端到端混为一谈。

---

## 6. 与论文 §5「最小可验证实验包」的对照（三条轴）

论文 `sec:eval-minimal` 的三条轴与仓库的**对应关系**（非一一等同，写作时措辞区分）：

| 轴 | 论文含义 | 仓库中主要支撑 |
|----|----------|----------------|
| (i) 信息经济学 | 多轮 F1–轮次、混淆/诱饵消融 | `multi_round_extraction` + 防御消融场景组合 |
| (ii) 价值锚定 | HPM、CoT、降级路径 | **部分**：`hpm_proxy`；**未全**：CoT 开关、完整 HPM |
| (iii) 表征异常 | 联合键 vs 纯语义、PBU | **未**：Track B / 嵌入实验；`rag_semantic_only` 仅为语义侧**客户端代理** |

---

## 7. 快速命令索引（复制前请读 `RUN_GUIDE.md` 全文）

- 正式核心（DeepSeek 示例）：`RUN_GUIDE.md` §2.4。  
- 全量多防御多种子：`PIPELINE_FORMAL_DATA.md` §3 或 `RUN_GUIDE.md` §2.7。  
- 真实上游 + 输出守卫环境模板：`RUN_GUIDE.md` §2.8。  
- 小样本有害/良性 + 守卫对比：`RUN_GUIDE.md` §2.9。  
- 环境变量全表：`README.md`「环境变量」与「论文实验数据」。

---

## 8. 修订与版本

- 评测协议升级时，以 `run_paper_benchmark.py` 内 **`PROTOCOL_VERSION`** 与 JSON **`manifest.protocol_version`** 为准。  
- 若本文与 `PIPELINE_FORMAL_DATA.md` / `RUN_GUIDE.md` 冲突，**以二者中更贴近命令行的描述为准**，并建议更新本 Playbook 本节日期。

---

*本文档旨在降低「从仓库到论文」的认知摩擦；具体数字的解释责任仍在于方法与附录中的协议披露。*
