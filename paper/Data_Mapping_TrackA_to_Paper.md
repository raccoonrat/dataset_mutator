# Track A 结果 JSON → 论文章节映射

本文档将仓库内**已有**评测 JSON 与《Beyond Model Reflection / 解耦安全》正文/附录的对应关系固定下来，便于写作与审稿核对。协议以 `manifest.protocol_version` 为准（当前制品为 **paper-eval-4**）。

**相关文件：**

- 主对齐说明：[DecoupledSafety_Empirical_Alignment.md](DecoupledSafety_Empirical_Alignment.md)
- 实验 Playbook：[../Decoupled-LLM-Gateway/experiments/EXPERIMENT_PLAYBOOK_CN.md](../Decoupled-LLM-Gateway/experiments/EXPERIMENT_PLAYBOOK_CN.md)

---

## 1. 提示数据版本（附录复现）

| 文件 | SHA256（全小写十六进制） | 非注释行数 |
|------|--------------------------|------------|
| `Decoupled-LLM-Gateway/experiments/data/harmful_prompts_trackA_en.txt` | `1394752d0697d354cbd75ed990dc7c0cdcc3e546da832d38a4f63e369a22a82c` | 8 |
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
