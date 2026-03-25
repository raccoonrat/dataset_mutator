# 论文实验目标与 Decoupled-LLM-Gateway 对齐说明

本文档汇总《Beyond Model Reflection / 解耦安全》**§5 评估协议**与仓库 **`Decoupled-LLM-Gateway`**（含 `paper-eval-4` 评测脚本）的对应关系，以及如何产出**可写入论文主表的 Track A 实证数据**。

**完整流水线（验证逻辑、正式数据获取步骤、数据流图）：**  
[`Decoupled-LLM-Gateway/experiments/PIPELINE_FORMAL_DATA.md`](../Decoupled-LLM-Gateway/experiments/PIPELINE_FORMAL_DATA.md)

---

## 1. 对齐关系（论文目标 → 代码）

| 论文 §5 目标 | 实现位置 |
|--------------|----------|
| 防御消融（统一 / 无混淆 / 无诱饵 / 意图-only） | 网关 `X-Gateway-Experiment-Mode`；脚本 `--defenses`：`unified`、`no_obfuscate`、`no_decoy`、`intent_only` |
| 单 guard、SmoothLLM 式扰动 | `direct_upstream`（直连 API 时脚本自动加 Bearer，密钥同 `DEEPSEEK_API_KEY` 等）+ echo 可用 `X-Echo-Refuse-Substr`；`smooth_llm` |
| 强系统提示 + 单 guard（§5 基线 2） | `strong_system_guard`（请求前插 system；**真实模型**上更有意义） |
| 语义 RAG、无联合键（§5 基线 3 的客户端代理） | `rag_semantic_only`（固定 exemplar 块，非嵌入检索） |
| RSR、单轮抽取 ASR / token-F1 | 场景 `refusal_keyword`、`extraction_leak` |
| §5 攻击 (7) 标准有害集 RSR | `harmful_rsr_suite` + `experiments/data/harmful_prompts_trackA_en.txt`；可 `fetch_advbench_subset.py` 扩展 |
| 可选 HTTP 裁判（审计对齐） | `--judge-mode http`、`PAPER_EVAL_JUDGE_URL` / `--judge-url`；`manifest.judge_mode` |
| **最小可验证包 (i)：多轮抽取 F1–轮次、达到 τ 的轮数** | `multi_round_extraction`（`rounds[]`、`rounds_to_f1_ge_tau`、`max_f1`） |
| Judge FPR 代理（held-out 良性集） | `benign_fpr_suite` + `Decoupled-LLM-Gateway/experiments/data/benign_prompts_en.txt` |
| Decoy-DoS / 良性 SLA 代理 | `decoy_dos_sla`（并发压力 + 良性 p50/p95） |
| 多种子 / 简易聚合 | `--seeds 42,43,…` → JSON 中 `aggregate_extraction_f1` |
| 真实模型写进论文主表 | `GATEWAY_UPSTREAM` 指向 OpenAI 兼容 API + `--openai-model <id>` 写入 `manifest` |

### 明确不在本制品内（须在论文中单独声明）

- **Track B**：GCG、表征对齐 PBU 等须独立实验环境。
- **HPM 专用基准**、CoT 开/关对照。
- **联合键 vs 纯语义**的**嵌入级**检索与下游修正（需向量服务与监控器，非当前脚本范围）。
- Worker 侧「每变体 **n≥5** 次采样」的**分布裁判**与置信区间更新策略。

---

## 2. 如何跑「可写进论文」的 Track A 数据

1. 将网关 **`GATEWAY_UPSTREAM`** 配置为真实 OpenAI 兼容端点，并设置 **`DEEPSEEK_API_KEY`**（或 **`GATEWAY_UPSTREAM_API_KEY`** / **`OPENAI_API_KEY`**）。DeepSeek 示例：`GATEWAY_UPSTREAM=https://api.deepseek.com`，请求体中 **`model`** 使用 `deepseek-chat`（详见 `Decoupled-LLM-Gateway/README.md`「DeepSeek」节）。
2. 建议 **`GATEWAY_ASYNC_LOG=0`**，避免网关 NDJSON 与评测脚本输出的 JSON 混在一起。
3. 启动 **`echo-llm`**（仅用于本地消融）或跳过 echo、只跑真实上游链路的网关。
4. 示例（完整套件 + 多防御 + 多种子）：

```bash
cd Decoupled-LLM-Gateway

python3 experiments/run_paper_benchmark.py --suite full \
  --defenses unified,no_obfuscate,no_decoy,intent_only,direct_upstream,smooth_llm,strong_system_guard,rag_semantic_only \
  --seeds 42,43,44,45,46 \
  --openai-model <你的模型名> \
  -o results/paper_eval_trackA.json
```

5. **论文引用**：在正文或附录中引用 `results/paper_eval_trackA.json` 的 **`manifest`**（协议版本、模型字段、URL 等）与 **`runs[*].results`**；并与论文附录表（评估矩阵 / 复现清单）中的 **模型 ID、随机种子、Track A** 声明一致。

6. **协议版本**：当前 harness 自述为 **`paper-eval-4`**（见脚本 docstring 与 JSON `manifest.protocol_version`）；含 **`judge_service/server.py`** 与 **`--smooth-llm-samples`**。

---

## 3. 论文与仓库中的对应修改（摘要）

| 位置 | 内容 |
|------|------|
| `paper/BeyondModelReflection_DecoupledSafety.tex` | 摘要中区分文献数字与 **Track A 可复现主指标**；新增 **「Reference Implementation and Empirical Validation」**（`sec:eval-artifact`）；可用性段提及 `Decoupled-LLM-Gateway/` |
| `paper/BeyondModelReflection_DecoupledSafety_CN.tex` | 同上中文版 |
| `Decoupled-LLM-Gateway/README.md` | 「论文实验数据」节与 `paper-eval-4`、场景表、真实 LLM 命令示例 |
| `Decoupled-LLM-Gateway/experiments/run_paper_benchmark.py` | `paper-eval-4`：有害 RSR、HTTP 裁判、SmoothLLM K 采样、多轮、HPM 代理、良性 FPR、Decoy-DoS/SLA、多种子聚合 |
| `Decoupled-LLM-Gateway/experiments/judge_service/server.py` | 本地 HTTP 裁判（启发式 / OpenAI moderation） |
| `Decoupled-LLM-Gateway/experiments/data/benign_prompts_en.txt` | 默认良性提示列表（可替换为更大 held-out 集） |
| CI | `make paper-eval-check` / `python3 experiments/run_paper_benchmark.py --self-check` |

---

## 4. 可选后续增强

若需支撑摘要中 **「同步路径 &lt;50 ms」** 类主张，可增设：

- 从网关 **NDJSON / Redis Stream** 读取 `sync_processing_ms`，在评测后处理脚本中汇总 **p50/p99**；或  
- 在 `run_paper_benchmark.py` 中增加对网关日志的解析步骤。

（当前 JSON 报告中的 `latency_ms` 主要为**端到端 HTTP 往返**，含上游推理时间，与「纯网关附加延迟」不同，写论文时需措辞区分。）

---

## 5. 相关路径速查

| 路径 | 说明 |
|------|------|
| `Decoupled-LLM-Gateway/experiments/run_paper_benchmark.py` | Track A 主评测入口 |
| `Decoupled-LLM-Gateway/experiments/data/benign_prompts_en.txt` | 良性 FPR 默认数据 |
| `paper/BeyondModelReflection_DecoupledSafety.tex` | 英文稿（含 `sec:eval-artifact`） |
| `paper/BeyondModelReflection_DecoupledSafety_CN.tex` | 中文稿 |

---

*文档与仓库实现同步维护；若协议版本升级，请同时更新 `PROTOCOL_VERSION` 与本文件中的版本号说明。*

**历史**：仓库内部分 `results/*.json` 仍为 `paper-eval-2` / `paper-eval-3` 留存，新跑数请以 `manifest.protocol_version == paper-eval-4` 为准。
