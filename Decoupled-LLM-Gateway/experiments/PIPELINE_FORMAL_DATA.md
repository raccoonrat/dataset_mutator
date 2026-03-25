# 代码如何验证论文 · 如何拿到正式实验数据（Pipeline）

本文说明 **`Decoupled-LLM-Gateway`** 与 **`run_paper_benchmark.py`（`paper-eval-4`）** 如何支撑《Beyond Model Reflection / 解耦安全》的实证部分，以及从环境到 **可写入论文的 JSON** 的完整流水线。

**更详尽的实验目标、能力边界与论文写作注意**：见同目录 **[`EXPERIMENT_PLAYBOOK_CN.md`](EXPERIMENT_PLAYBOOK_CN.md)**。

---

## 一、验证关系：论文主张 ←→ 代码在测什么

| 论文中的内容 | 代码/脚本如何支撑验证 |
|--------------|------------------------|
| **解耦安全架构**（同步：混淆 + 诱饵 + 策略降级；异步：日志流 + Worker + 策略回灌） | `cmd/gateway`、`worker/main.py`、Redis 实现可部署切片；**正式表数据**主要来自经网关调用真实模型时脚本汇总的 **RSR / ASR / F1 / 延迟 / 多轮曲线** 等。 |
| **§5 Track A**（黑盒、提示级攻击） | 脚本默认 `track=A`；不把需白盒的指标与 Track A 主表混写。 |
| **防御消融** | `X-Gateway-Experiment-Mode`：`unified` / `no_obfuscate` / `no_decoy` / `intent_only`。 |
| **基线对照** | `direct_upstream`（直连 API，脚本自动加 `Authorization: Bearer`，密钥同网关：`DEEPSEEK_API_KEY` 等）、`smooth_llm`、`strong_system_guard`、`rag_semantic_only`。 |
| **RSR** | `refusal_keyword` + 回复文本规则判定（真实模型下为主指标之一）。 |
| **抽取 ASR、token-F1** | `extraction_leak`、`multi_round_extraction`（真实模型下无 echo 的确定性泄漏，数值反映真实合规性）。 |
| **信息经济学代理（轮数 × 质量）** | `multi_round_extraction`：`rounds[]`、`max_f1`、`rounds_to_f1_ge_tau`（写作时固定 `N_max`、τ、预算）。 |
| **良性 FPR 代理** | `benign_fpr_suite` + `data/benign_prompts_en.txt`（可换更大 held-out 列表）。 |
| **Decoy-DoS / SLA 代理** | `decoy_dos_sla`：并发压力后良性成功率与 p95 延迟。 |
| **可复现性** | JSON `manifest`：`protocol_version`、`openai_model_field`、URL；`GIT_SHA`；多种子 `runs[]` 与 `aggregate_extraction_f1`。 |

**本制品不覆盖（论文需另述或引用）**：Track B（GCG 等）、HPM 专用集与 CoT 消融、嵌入级联合键、Worker 内 n≥5 次采样的分布裁判。

---

## 二、「正式数据」与「烟测数据」

| 类型 | 上游 | 用途 |
|------|------|------|
| **烟测** | `echo-llm` | 验证网关与头行为、CI `--self-check`；**不作论文主表**。 |
| **正式数据** | **DeepSeek / OpenAI 兼容 API** | 主文与附录表格；须记录模型名、日期、`temperature`、种子、**原始 JSON**。 |

正式数据最低要求：

1. 网关：`GATEWAY_UPSTREAM=https://api.deepseek.com` + `DEEPSEEK_API_KEY`（或 `GATEWAY_UPSTREAM_API_KEY`）。  
2. 脚本：`--openai-model deepseek-chat`（写入 `manifest` 与请求体）。  
3. 多种子：`--seeds 42,43,44,45,46` 或固定单种子并在文中声明。  
4. 归档：`results/*.json`（脱敏后入制品库或附录）。  
5. 正文引用 **`manifest.protocol_version`**（当前 **`paper-eval-4`**）、**`manifest.judge_mode`**、**`manifest.smooth_llm_samples_k`** / **`smooth_llm_sigma`**（若使用 SmoothLLM 采样）、**`manifest.gateway_output_guard_header`**（若使用 `--gateway-output-guard`）。

**有害单轮 RSR 数据**：默认 `experiments/data/harmful_prompts_trackA_en.txt`；扩展可用 `experiments/scripts/fetch_advbench_subset.py` 生成列表，评测时 `--harmful-prompts-file <path>`。

**HTTP 裁判**：`--judge-mode http` + `PAPER_EVAL_JUDGE_URL`（或 `--judge-url`）；可选 `PAPER_EVAL_JUDGE_BEARER`。本地实现 `judge_service/server.py`：`JUDGE_BACKEND=heuristic` | `openai_moderation` | `chat_completion`（OpenAI 兼容 chat，YES/NO 解析；`JUDGE_CHAT_PROMPT_TEMPLATE` 可覆写模板）。自检：`python3 experiments/judge_service/server.py --self-check`（亦由 `run_paper_benchmark.py --self-check` 间接调用）。

**网关可选输出守卫**（与论文主表无强绑定，属实现增强）：配置 `GATEWAY_OUTPUT_GUARD_URL` 等（见 `README.md` 环境变量表）。**默认**仅当请求含 **`X-Gateway-Output-Guard: 1`** 时才调用守卫；评测脚本对非 `direct_upstream` 路径可加 **`--gateway-output-guard`** 统一带头。**风险**：将「非拒绝」映射为「用模板覆盖」会伤害真实良性对话，生产或 FPR 实验须谨慎或设 `GATEWAY_OUTPUT_GUARD_REQUIRE_HEADER=0` 仅在隔离环境使用。

**本地烟测（无外部 LLM）**：`make smoke-output-guard` 或 `bash experiments/scripts/smoke_output_guard.sh`（echo + 启发式 `judge_service` + 网关），验证三条路径：无头、带头→模板、带头+echo 拒答→保留原文。

**真实上游 + 守卫 + 小样本 A/B**：`--max-harmful-prompts` / `--max-benign-fpr-prompts` 控制费用；同一命令加/不加 `--gateway-output-guard` 各跑一次，核对 `manifest.gateway_output_guard_header` 与 `per_prompt`。见 `RUN_GUIDE.md` **§2.9** 与 `experiments/scripts/run_real_upstream_guard_compare.sh`。

---

## 三、正式实验 Pipeline（推荐顺序）

### 0. 固定版本（可选）

```bash
cd Decoupled-LLM-Gateway
go build -o bin/gateway ./cmd/gateway
export GIT_SHA=$(git rev-parse HEAD)
```

### 1. 启动网关 → 真实模型

```bash
export DEEPSEEK_API_KEY='你的密钥'
export GATEWAY_UPSTREAM='https://api.deepseek.com'
export GATEWAY_ASYNC_LOG=0
export GATEWAY_UPSTREAM_TIMEOUT_MS=120000
./bin/gateway
```

日志应含 **`upstream_auth=bearer`**（不打印密钥）。

### 2. 运行评测脚本 → 原始 JSON

**核心 Track A（经网关，推荐先跑）：**

```bash
mkdir -p results
python3 experiments/run_paper_benchmark.py \
  --gateway-url http://127.0.0.1:8080 \
  --upstream-url https://api.deepseek.com \
  --openai-model deepseek-chat \
  --defenses unified,no_obfuscate,no_decoy,intent_only \
  --scenarios benign_baseline,refusal_keyword,extraction_leak \
  --seed 42 \
  -o results/trackA_core_seed42.json
```

**含直连基线 `direct_upstream`：** 脚本对 `https://api.deepseek.com` 的调用会读取 **`DEEPSEEK_API_KEY`**（或 `GATEWAY_UPSTREAM_API_KEY` / `OPENAI_API_KEY`）自动加 Bearer，**无需**再经网关。

**论文级全量（多防御 + `--suite full` + 多种子）：**

```bash
export DEEPSEEK_API_KEY='…'   # direct_upstream 与网关共用

python3 experiments/run_paper_benchmark.py --suite full \
  --gateway-url http://127.0.0.1:8080 \
  --upstream-url https://api.deepseek.com \
  --openai-model deepseek-chat \
  --defenses unified,no_obfuscate,no_decoy,intent_only,direct_upstream,smooth_llm,strong_system_guard,rag_semantic_only \
  --seeds 42,43,44,45,46 \
  -o results/trackA_full_paper_eval4.json
```

（文件名示例；以 `manifest.protocol_version == paper-eval-4` 为准。）

### 3. 从 JSON 到论文章节

1. 用 `jq`/Notebook 汇总 `runs[].results[]` 中各 `scenario_id` × `defense_baseline` 的指标。  
2. 多轮：`multi_round_extraction` 的 `rounds` / `max_f1` / `rounds_to_f1_ge_tau` 做表或曲线。  
3. 附录贴 **`manifest`** 与运行命令；敏感字段脱敏。  
4. 摘要/结论只陈述 **本 harness + 本模型** 下可支持的命题，文献数字仍标注引用。

### 4. 异步闭环（可选，非脚本必填）

若论文需 **「诱饵泄露 → Worker 写策略 → 二次请求降级」** 的端到端叙事：`docker compose up redis`、起 `worker/main.py`、网关配 `GATEWAY_REDIS_ADDR`；主表仍以 **§5 脚本指标** 为主，闭环作 **系统演示**。

---

## 四、数据流示意图

```
[评测脚本] --HTTP--> [网关 :8080] --混淆/诱饵/策略--> [DeepSeek API]
                         |
                    （可选）Redis Stream --> Worker --> 策略回灌

direct_upstream 基线：[评测脚本] --Bearer--> [DeepSeek API]（不经网关）
```

---

## 五、相关文件

| 路径 | 说明 |
|------|------|
| `experiments/run_paper_benchmark.py` | 主入口，`paper-eval-4` |
| `experiments/judge_service/server.py` | 可选：本地 HTTP 裁判 |
| `experiments/data/benign_prompts_en.txt` | 默认良性集 |
| `experiments/examples/deepseek_sdk_smoke.py` | 不经网关的 SDK 烟测 |
| `../README.md` | DeepSeek 环境变量与网关说明 |
| `../../paper/DecoupledSafety_Empirical_Alignment.md` | 与 LaTeX `sec:eval-artifact` 对齐摘要 |

---

*随 `PROTOCOL_VERSION` 升级请更新本节中的版本号说明。*
