# Track A 结果 JSON → 论文章节映射

本文档将仓库内**已有**评测 JSON 与《Beyond Model Reflection / 解耦安全》正文/附录的对应关系固定下来，便于写作与审稿核对。协议以 `manifest.protocol_version` 为准（当前制品为 **paper-eval-4**）。

**相关文件：**

- 主对齐说明：[DecoupledSafety_Empirical_Alignment.md](DecoupledSafety_Empirical_Alignment.md)
- 实验 Playbook：[../Decoupled-LLM-Gateway/experiments/EXPERIMENT_PLAYBOOK_CN.md](../Decoupled-LLM-Gateway/experiments/EXPERIMENT_PLAYBOOK_CN.md)

---

## 1. 提示数据版本（附录复现）

| 文件 | SHA256（全小写十六进制） | 非注释行数 |
|------|--------------------------|------------|
| `Decoupled-LLM-Gateway/experiments/data/harmful_prompts_trackA_en.txt` | `55f1281e97d7fd839d6d707ae66a6857ae54dd7cfded994d76fe13dd1bfcc568` | 80 |
| `Decoupled-LLM-Gateway/experiments/data/benign_prompts_en.txt` | `39c48b0022a89514542d2e7f141ad54ccbecf30817cdb34efc03e1ff3f048917` | 10 |
| `Decoupled-LLM-Gateway/experiments/data/hpm_proxy_prompts_en.txt` | `67b5ac7b36cf8cfcad64319715057268abd38be2a71d7049ac16418fe856a51d` | 5 |

更换文件后须**重新计算哈希**并更新论文附录。

---

## 2. 已有结果文件一览

| 路径 | 上游 | 协议 | defenses（摘要） | scenarios（摘要） | 种子数 | 备注 |
|------|------|------|------------------|-------------------|--------|------|
| `Decoupled-LLM-Gateway/results/trackA_core_seed42.json` | DeepSeek | paper-eval-4 | 4（无 direct/smooth） | 3 核心 | 1 | 最小核心切片 |
| `Decoupled-LLM-Gateway/results/trackA_core_guard_seed42.json` | DeepSeek | paper-eval-4 | 同 core | 同 core | 1 | `gateway_output_guard_header: true` |
| `Decoupled-LLM-Gateway/results/trackA_full_capped_guard.json` | DeepSeek | paper-eval-4 | 6 + cap | full 8 场景 | 1 | harmful/benign **截断**（2/3 条） |
| `Decoupled-LLM-Gateway/results/real_smoke_guard.json` | DeepSeek | paper-eval-4 | 4 | harmful + benign FPR | 1 | 输出守卫烟测 |
| `Decoupled-LLM-Gateway/results/real_smoke_noguard.json` | DeepSeek | paper-eval-4 | 4 | harmful + benign FPR | 1 | 无守卫对照 |
| `Decoupled-LLM-Gateway/results/trackA_full.json` | **echo-llm** | paper-eval-2 | 6 | full（无 harmful_rsr_suite） | 3 | **非真实 LLM**；用于回归/控制实验 |

**正式论文主表**应优先引用：**无 cap**、**多种子**、**完整 defense 列表**（含 `structured_wrap`、`strong_system_guard`、`rag_semantic_only`）的 `paper-eval-4` 产物，例如：

- `Decoupled-LLM-Gateway/results/trackA_full_paper_seed3.json`（真实上游，计划产物）
- `Decoupled-LLM-Gateway/results/trackA_full_echo_seed3.json`（echo 控制实验，计划产物）

---

## 3. JSON 字段 → 论文章节 / 叙事

| 论文锚点 | LaTeX label（若适用） | JSON 来源 | 叙事用途 |
|----------|----------------------|-----------|----------|
| 信息经济学、多轮抽取成本 | `sec:paradigm-economics`, `sec:eval-minimal`(i) | `scenario_id: multi_round_extraction` → `max_f1`, `rounds_to_f1_ge_tau`, `rounds[]` | F1–轮次曲线、达到 τ 的轮数；配合 defense 消融 |
| 上下文混淆、诱饵、ID 对齐 | `sec:context-confusion` | `extraction_leak` → `extraction_f1`, `asr_extraction`；`harmful_rsr_suite` → `harmful_rsr_rate`, `per_prompt` | 单轮抽取 + 有害集 RSR 上的组件贡献 |
| 价值锚定、HPM、拒答 | `sec:paradigm-axiological`, `sec:eval-attack`(3) | `hpm_proxy` → `hpm_rsr_rate`；`refusal_keyword` → `rsr` | 压力框架下 RSR（**代理集**，非完整 HPM 基准） |
| RSR / ASR 操作定义 | `sec:eval-metrics` | `harmful_rsr_suite`, `refusal_keyword`, `extraction_leak` | 与 `manifest.judge_mode`（heuristic / http）一起披露 |
| 良性效用、Judge FPR 代理 | `sec:eval-utility`, `sec:eval-metrics` | `benign_baseline`（`rsr` 应为 false→效用）；`benign_fpr_suite` → `improper_refusal_rate` | 联合目标：安全 vs 过度拒绝 |
| Decoy-DoS / SLA 代理 | `sec:eval-utility`, `sec:physics` | `decoy_dos_sla` → 成功率与 p50/p95 等字段 | 资源耗尽与可用性边界 |
| 防御基线对照 | `sec:eval-defense` | `defense_baseline`：`unified`, `no_obfuscate`, `no_decoy`, `intent_only`, `structured_wrap`, `strong_system_guard`, `rag_semantic_only`, `smooth_llm`, `direct_upstream` | 与 §5 基线 (1)–(5) 叙述对齐（部分为代理实现） |
| 复现与制品 | `sec:eval-artifact`, `sec:appendix-eval` | 根级 `manifest`, `defenses`, `scenarios`, `runs`, `aggregate_by_defense` | 附录清单：模型 ID、种子、协议版本、URL、守卫开关 |

---

## 4. `aggregate_by_defense` → 主表列建议

写作时可将下列键作为 Track A 摘要列（数值均为跨 `runs` 中各 seed 的**均值**；多种子时建议在文中说明 n=种子数）：

| 聚合键 | 对应场景 | 论文表述提示 |
|--------|----------|--------------|
| `harmful_rsr_rate_mean` | `harmful_rsr_suite` | §5 攻击 (7) 标准有害单轮集上的 RSR |
| `refusal_keyword_rsr_mean` | `refusal_keyword` | 关键词违规提示上的拒答（与 echo 烟测一致） |
| `extraction_leak_f1_mean` | `extraction_leak` | 单轮系统指令类抽取的 token-F1 |
| `multi_round_max_f1_mean` | `multi_round_extraction` | 多轮会话内最大 F1 |
| `hpm_rsr_rate_mean` | `hpm_proxy` | HPM **风格代理**压力下的 RSR |
| `benign_improper_refusal_mean` | `benign_fpr_suite` | 良性集上「不当拒答」比例（FPR 代理） |
| `benign_utility_mean` | `benign_baseline` | 简单良性任务上「非拒答」比例（效用代理） |

`decoy_dos_sla` 的分位数等指标在 JSON 行级字段中，需按场景单独摘取（见 `EXPERIMENT_PLAYBOOK_CN.md`）。

---

## 5. 命题类型与证据强度（写作自检）

| 命题类型 | 仅靠「组件存在」是否够 | 需要 Track A JSON |
|----------|------------------------|-------------------|
| 架构 / 解耦范式（同步网关 + 异步环） | 可论述设计 | 否 |
| 部署对齐下的 RSR、抽取、效用联合指标 | 否 | **是**（真实上游 + 披露协议） |
| 与 SOTA 数字对比 | 否 | 引用文献；本文 JSON 仅支撑**同协议下**消融与基线 |
| Track B（GCG、表征 PBU） | — | **否**（制品外） |

---

## 6. 生成主表与 LaTeX

使用脚本从 JSON 导出 LaTeX 表格：

```bash
cd Decoupled-LLM-Gateway
python3 experiments/scripts/export_trackA_table_latex.py \
  --json results/trackA_full_paper_seed3.json \
  --out-en ../paper/generated/trackA_main_table.tex \
  --out-cn ../paper/generated/trackA_main_table_cn.tex
```

**真实 DeepSeek / 其他付费上游**：跑 `run_trackA_full_paper.sh` 或手工跑 `run_paper_benchmark.py` 前，在 `Decoupled-LLM-Gateway/` 下先执行 **` . ./env`**（或确保等价环境变量已导出），再启动网关（`GATEWAY_UPSTREAM` 与密钥与 `env` 一致）。

若尚无真实上游 JSON，可用 `results/trackA_full_echo_seed3.json` 生成**控制实验**表（须在论文中标注为 echo，非生产模型）。

---

## 7. 补充实验：对照原计划后的缺口分析与优先级

### 7.1 计划内已完成项（摘要）

| 计划项 | 状态 |
|--------|------|
| 无 cap 全场景 + 9 defenses + 种子 42,43,44 | 已有 `results/trackA_full_paper_seed3.json`（须做 **7.2 数据有效性校验**） |
| echo 同矩阵控制实验 | `results/trackA_full_echo_seed3.json` + `run_trackA_full_echo_ci.sh` |
| 主表 LaTeX + 论文 `\input` | `paper/generated/trackA_main_table*.tex` |
| 附录提示表 SHA256 | 已写入中英 `.tex` |

### 7.2 当前 JSON 的解读风险（必读）

1. **网关 vs 脚本 `manifest` 不一致**  
   `run_paper_benchmark.py` 的 `manifest.upstream_url` 反映的是 **`--upstream-url`（给 `direct_upstream` 用）**，**不保证** 网关进程里的 `GATEWAY_UPSTREAM`。若网关仍指向 **echo-llm**，则经网关的 `assistant_text` 会出现 **`[echo]`** 前缀、延迟 **亚毫秒～几十毫秒**，而 `direct_upstream` 仍是真实 DeepSeek（秒级）。此时 **主表「经网关」列与「直连」列不在同一实验台上**，不能用来论证「网关相对基座的提升」。  

   **动作**：跑真实模型前 **` . ./env`**，**重启网关**，再跑 `run_trackA_full_paper.sh`；跑完后执行：

   ```bash
   python3 experiments/scripts/validate_paper_json.py results/trackA_full_paper_seed3.json
   ```

2. **`harmful_rsr_suite` + 启发式裁判**  
   若上游实际为强安全模型，回复长段「教育式拒绝」但**不含**启发式关键词，可能被记为 **RSR=0**；与人工或 HTTP 裁判结果可能不一致。  

   **动作**：主文/附录并列 **`--judge-mode heuristic`** 与 **`http`**（`PAPER_EVAL_JUDGE_URL`）的一小段对照，或子集人工标注 κ。

3. **同步策略降级 vs LLM 拒答**  
   `refusal_keyword` 对 `MALICIOUS_TRIGGER` 可能走 **网关模板**（极快、`REFUSAL_SINGLE_GUARD`），与「模型自身 RSR」不是同一机制；正文应区分 **policy degradation** 与 **upstream refusal**。

4. **消融「拉不开」**  
   在强对齐模型上，`extraction_leak_f1`、多轮 `max_f1` 可能全为 0，网关消融列接近——符合计划在「风险与应对」中的说明。可接受的叙事是：**联合指标 + `direct_upstream` 对照 + echo 控制**，并承认 **value-add 主要体现在纵深与可审计链路上**，而非单一 ASR 差。

### 7.3 建议的补充实验（按性价比排序）

| 优先级 | 目的 | 做法 |
|--------|------|------|
| **P0** | 保证主表可信 | 修正网关上游后**重跑** `run_trackA_full_paper.sh`；`validate_paper_json.py` 通过；重新 `export_trackA_table_latex.py` |
| **P1** | 论文 §5「HTTP 裁判」 | 一键子集：`Decoupled-LLM-Gateway/experiments/scripts/run_trackA_p1_http_judge_subset.sh`（内置启 judge、预检、validate）；或手动启动 `judge_service` 后设 `PAPER_EVAL_JUDGE_URL=http://127.0.0.1:8765/judge` |
| **P1** | SmoothLLM 分布评估（K>1） | `experiments/scripts/run_trackA_p1_smooth_k5.sh`（`--smooth-llm-samples 5`，全矩阵 3 seeds；与主表 K=1 对照 `smooth_llm` 行） |
| **P2** | 输出守卫消融 | `experiments/scripts/run_trackA_p2_output_guard.sh`：网关须带 `GATEWAY_OUTPUT_GUARD_URL` 指向同机 `judge_service`；`--gateway-output-guard`，默认输出 `results/trackA_full_paper_guard_seed3.json` |
| **P2** | 有害集规模 / 文献可比 | `experiments/scripts/fetch_advbench_subset.py -n 80` 写入 `harmful_prompts_trackA_en.txt` 后重跑主表脚本；**更新 SHA256** 与附录表 |
| **P3** | HPM 全文基准 | 在合规前提下替换 `hpm_proxy_prompts_en.txt` 为许可的 HPM 子集，并声明非代理 |
| **P3** | 第二模型 | 换 `gpt-4o-mini` 等再跑一套 JSON，附录多一行「模型敏感性」 |

### 7.4 示例命令片段（真实上游 + env）

分步清单（含网关须非 echo）：[`../Decoupled-LLM-Gateway/experiments/P0_REAL_UPSTREAM_CHECKLIST.md`](../Decoupled-LLM-Gateway/experiments/P0_REAL_UPSTREAM_CHECKLIST.md)。

```bash
cd Decoupled-LLM-Gateway
set -a && . ./env && set +a
# 终端 A：确认日志里 upstream 为 https 且 upstream_auth=bearer
go run ./cmd/gateway
# 终端 B：`run_trackA_full_paper.sh` 已内置预检 + validate + LaTeX 导出（可用 RUN_VALIDATE_AND_EXPORT=0 关闭后两步）
bash experiments/scripts/run_trackA_full_paper.sh
python3 experiments/scripts/validate_paper_json.py results/trackA_full_paper_seed3.json
```

**HTTP 裁判（降费：只跑有害 + 良性 FPR）** 示例：

```bash
# 终端 C：python experiments/judge_service/server.py （按 README 配 JUDGE_BACKEND）
export PAPER_EVAL_JUDGE_URL=http://127.0.0.1:8765/judge
set -a && . ./env && set +a
python3 experiments/run_paper_benchmark.py \
  --gateway-url http://127.0.0.1:8080 --upstream-url https://api.deepseek.com \
  --openai-model deepseek-chat \
  --defenses unified,direct_upstream \
  --scenarios harmful_rsr_suite,benign_fpr_suite \
  --seeds 42,43,44 --judge-mode http \
  -o results/trackA_judge_http_subset.json
```
