# Decoupled-LLM-Gateway 运行指南

本文说明两种上游：**本地 echo-llm（烟测）** 与 **真实大模型 API（正式评测）**，以及如何跑通 `curl` 与 `run_paper_benchmark.py`。

**仓库根目录**：始终在 `Decoupled-LLM-Gateway/` 下执行命令（或把路径换成你的克隆位置）。

---

## 通用准备

```bash
cd Decoupled-LLM-Gateway
# Go 1.21+；Python 3 用于评测脚本
```

| 组件 | 默认地址 | 说明 |
|------|-----------|------|
| **网关** | `http://127.0.0.1:8080` | 客户端只打这里（经网关场景） |
| **echo 上游** | `http://127.0.0.1:9090` | Mock OpenAI 兼容 `/v1/chat/completions` |
| **健康检查** | `GET http://127.0.0.1:9090/healthz` | echo 就绪后再起网关更稳 |

---

## 一、上游：echo-llm（烟测 / 消融 / 低成本验证）

### 1.1 启动方式（二选一）

**方式 A — 两个终端**

```bash
# 终端 1
make run-echo

# 终端 2（上游指向 echo，默认已是 9090）
make run-gateway
```

**方式 B — 一键（后台 echo + 前台网关）**

```bash
bash scripts/dev.sh
```

### 1.2 快速验证链路

```bash
curl -sS http://127.0.0.1:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"eval","messages":[{"role":"user","content":"hello"}]}'
```

应返回 JSON，且 `choices[0].message.content` 为 echo 合成内容。

### 1.3 跑论文评测脚本（经网关）

echo **不需要** API Key。`direct_upstream` 基线会直连 `9090`，其余 defense 走 `8080`。

```bash
mkdir -p results

python3 experiments/run_paper_benchmark.py \
  --gateway-url http://127.0.0.1:8080 \
  --upstream-url http://127.0.0.1:9090 \
  --openai-model eval \
  --defenses unified,no_obfuscate,no_decoy,intent_only \
  --scenarios benign_baseline,refusal_keyword,extraction_leak \
  --seed 42 \
  -o results/trackA_echo_seed42.json
```

**说明**：echo 可通过请求头 `X-Echo-Refuse-Substr`、`X-Echo-Leak-System`、`X-Echo-Eval-Secret` 等控制拒答/泄漏行为；脚本在对应场景下会自动带这些头。此类 JSON **用于流水线与协议自检**，**不作为论文主表**（主表需真实模型）。

**可选：网关输出守卫 + 评测带头**（与 `README.md` 环境变量表一致）：若网关配置了 `GATEWAY_OUTPUT_GUARD_URL`，上游 200 且外部裁判判定「非拒绝」时可能用模板覆盖正文；**默认仅在请求带 `X-Gateway-Output-Guard: 1` 时启用**（避免对普通良性流量误杀）。对经网关的 defense（**不含** `direct_upstream`），可加：

`--gateway-output-guard`

使脚本自动附加该头；`manifest.gateway_output_guard_header` 会记录是否开启。**良性 FPR / 日常对话场景请勿随意开启**，除非明确在做受控红队或你已理解「非拒绝→覆盖」语义。

### 1.4 仅测脚本逻辑（不启 echo/网关）

```bash
make paper-eval-check
# 等价：python3 experiments/run_paper_benchmark.py --self-check
```

### 1.5 输出守卫端到端烟测（echo + `judge_service` + 网关）

在**不调用**外部大模型 API 的前提下，验证：`GATEWAY_OUTPUT_GUARD_URL` 指向 `judge_service` 时，无 `X-Gateway-Output-Guard` 则不过守卫；带头则对「非拒绝」回复按模板覆盖；echo 拒答路径下保留上游拒答正文。

```bash
make smoke-output-guard
# 等价：bash experiments/scripts/smoke_output_guard.sh
```

脚本默认用**三个临时端口**（避免占用 8080/9090）；可通过 `SMOKE_ECHO_PORT`、`SMOKE_JUDGE_PORT`、`SMOKE_GW_PORT` 固定端口。通过后，可将同一 `GATEWAY_OUTPUT_GUARD_URL` 配入日常网关，并对评测流量加 `run_paper_benchmark.py --gateway-output-guard`。

---

## 二、上游：真实大模型 API（DeepSeek / OpenAI 兼容）

### 2.1 密钥与上游地址

网关向上游发 `POST {GATEWAY_UPSTREAM}/v1/chat/completions`，并在配置了密钥时加：

`Authorization: Bearer <token>`

**密钥优先级**（任设其一即可）：

1. `GATEWAY_UPSTREAM_API_KEY`（推荐）
2. `DEEPSEEK_API_KEY`
3. `OPENAI_API_KEY`

### 2.2 启动网关（以 DeepSeek 为例）

```bash
export DEEPSEEK_API_KEY='你的密钥'
export GATEWAY_UPSTREAM='https://api.deepseek.com'
export GATEWAY_ASYNC_LOG=0
export GATEWAY_UPSTREAM_TIMEOUT_MS=120000
go run ./cmd/gateway
```

日志中应出现 **`upstream_auth=bearer`**（**不会**打印密钥）。若出现 **`upstream_auth=off`** 且上游是 `https://api.deepseek.com` 等公网地址，说明**当前进程没有读到** `GATEWAY_UPSTREAM_API_KEY` / `DEEPSEEK_API_KEY` / `OPENAI_API_KEY`——常见于：只在「另一个终端」或 `~/.bashrc` 里写了变量但未 `export`、或启动网关的 shell 与设 key 的 shell 不是同一个。请在该终端执行 `export DEEPSEEK_API_KEY=...`（或 `GATEWAY_UPSTREAM_API_KEY=...`）后**再** `go run ./cmd/gateway`；启动时若仍缺 key，网关会打印一行 **warning** 提示。若需同步日志顺序调试，可设 `GATEWAY_LOG_AFTER_RESPONSE=0`（见 `README.md` 环境变量表）。

**必读（易踩坑）**：`run_paper_benchmark.py` 的 **`--upstream-url` 不会修改网关进程**。它只告诉脚本：当 `defenses` 里包含 **`direct_upstream`** 时，**直连**哪一个 OpenAI 兼容根 URL。  
凡 **`unified` / `no_obfuscate` / `intent_only` 等经网关的 defense**，请求发往 **`--gateway-url`**，由**已启动的网关**按环境变量 **`GATEWAY_UPSTREAM`** 转发。若未设置 `GATEWAY_UPSTREAM`，网关默认 **`http://127.0.0.1:9090`（echo）**；echo 未启动则返回 **502**，`assistant_text` 里可见 `dial tcp 127.0.0.1:9090: connect: connection refused`。  
因此：跑真实 API 时，**先**在同一 shell（或 systemd 环境）里 `export GATEWAY_UPSTREAM='https://api.deepseek.com'` 与密钥，**再** `go run ./cmd/gateway`，并与 §2.4 的 `--upstream-url` **保持一致**。

### 2.3 快速验证（经网关）

```bash
curl -sS http://127.0.0.1:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"deepseek-chat","messages":[{"role":"user","content":"用一句话说你好"}]}'
```

### 2.4 跑论文评测脚本（经网关 + 真实模型）

`--openai-model` 写入请求体与 JSON `manifest`，须与服务商模型 id 一致（如 `deepseek-chat`）。

**`--upstream-url`** 必须与 **`GATEWAY_UPSTREAM`（网关实际转发地址）相同**（供 `direct_upstream` 直连对比）；脚本会从环境变量读 Bearer。  
**仅跑 `unified` 等、不含 `direct_upstream` 时**：也仍须把网关配成真实上游，否则经网关的请求仍会打到默认 **9090**。

```bash
export DEEPSEEK_API_KEY='你的密钥'
export GATEWAY_UPSTREAM='https://api.deepseek.com'   # 与启动网关的终端一致；勿只写在评测命令里
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

上述 `export GATEWAY_UPSTREAM` 是给**与评测同一环境、用于启动网关**时对照用的；若网关在**另一终端**已运行，必须在**那个终端**里先 `export GATEWAY_UPSTREAM=...` 再启动 `go run ./cmd/gateway`，改完后需**重启网关**。

**全量 + 多种子**（耗时长、费用高）示例：

```bash
python3 experiments/run_paper_benchmark.py --suite full \
  --gateway-url http://127.0.0.1:8080 \
  --upstream-url https://api.deepseek.com \
  --openai-model deepseek-chat \
  --defenses unified,no_obfuscate,no_decoy,intent_only,direct_upstream,smooth_llm \
  --seeds 42,43,44 \
  -o results/trackA_full.json
```

### 2.5 OpenAI 官方 API

将 `GATEWAY_UPSTREAM` 设为 `https://api.openai.com/v1`，使用 `OPENAI_API_KEY`，`--openai-model` 设为例如 `gpt-4o-mini`，其余与上类似。

### 2.6 可复现性（建议）

```bash
export GIT_SHA="$(git rev-parse HEAD)"
export EVAL_HOSTNAME="$(hostname)"
```

评测 JSON 的 `manifest` 会收录上述字段（若脚本已支持读取环境变量）。

### 2.7 论文级套件与裁判（paper-eval-4 摘要）

- **`--suite full`**：在 `--scenarios` 基础上追加 `multi_round_extraction`、`harmful_rsr_suite`、`hpm_proxy`、`benign_fpr_suite`、`decoy_dos_sla`。有害提示默认 `experiments/data/harmful_prompts_trackA_en.txt`；可用 `experiments/scripts/fetch_advbench_subset.py` 从公开 CSV 拉取子集再 `--harmful-prompts-file` 指向生成文件。
- **裁判**：默认 `--judge-mode heuristic`（关键词）。审计对齐用 `--judge-mode http`，并设 `PAPER_EVAL_JUDGE_URL`（或 `--judge-url`）；本地服务见 `experiments/judge_service/server.py`（`JUDGE_BACKEND=heuristic` / `openai_moderation` / `chat_completion`，后者配合 `JUDGE_CHAT_BASE_URL`、`JUDGE_CHAT_MODEL` 等）。详情与契约见 `README.md`「论文实验数据」节与 `experiments/benchmark_spec_trackA.json`。
- **SmoothLLM 式 K 次采样**：`--smooth-llm-samples K`、`--smooth-llm-sigma σ`，**仅**在 `--defenses` 含 `smooth_llm` 时 K>1 生效；`manifest.smooth_llm_samples_k` / `smooth_llm_sigma` 写入 JSON。
- **多种子**：`--seeds 42,43,44` → `runs[]` 与 `aggregate_by_defense`（含有害 RSR、HPM、不当拒绝率等聚合字段）。

### 2.8 真实上游 + 输出守卫 + 最小场景（端口与环境模板）

**常见错误**：只执行了 `go run ./cmd/gateway`（或 `make run-gateway`），**没有**先 `export GATEWAY_UPSTREAM='https://api.deepseek.com'`，日志里会出现 `upstream http://127.0.0.1:9090`；此时跑 §2.4 / 本节评测会得到 **502**，JSON 中 `assistant_text` 含 **`connection refused`** 连 **9090**。解决办法：按下面「环境变量模板」**重启网关**；或另开终端 **`make run-echo`** 让 9090 有服务（仅本地烟测）。  

在 **真实 OpenAI 兼容上游** 上同时打开 **网关输出守卫** 时：网关会在上游返回 200 后，向 `GATEWAY_OUTPUT_GUARD_URL` 再发一次 `refusal_binary` 请求；若裁判判定 **非拒绝**，可用模板覆盖完成体。**默认**在配置了守卫 URL 时，仅当客户端带 **`X-Gateway-Output-Guard: 1`** 才调用守卫（避免误伤普通流量）。下面用固定端口便于照抄；可按需改端口并全局替换 URL。

| 角色 | 地址（示例） | 说明 |
|------|----------------|------|
| **HTTP 裁判** | `http://127.0.0.1:8765` | `judge_service` 监听；POST 路径任意，示例用 `/judge` |
| **网关** | `http://127.0.0.1:8080` | 评测脚本 `--gateway-url`、日常 `curl` 只打这里 |
| **真实上游** | `https://api.deepseek.com` 等 | 无本地端口；由 `GATEWAY_UPSTREAM` 指定 |

**环境变量模板（终端 B：网关）**——在 `Decoupled-LLM-Gateway/` 下执行；将密钥与上游换成你的服务商。

```bash
# --- DeepSeek 示例 ---
export DEEPSEEK_API_KEY='你的密钥'
export GATEWAY_LISTEN=':8080'
export GATEWAY_UPSTREAM='https://api.deepseek.com'
export GATEWAY_ASYNC_LOG=0
export GATEWAY_UPSTREAM_TIMEOUT_MS=120000

# 输出守卫：指向本机 judge_service（须与下方启动端口一致）
export GATEWAY_OUTPUT_GUARD_URL='http://127.0.0.1:8765/judge'
export GATEWAY_OUTPUT_GUARD_TIMEOUT_MS='15000'
export GATEWAY_OUTPUT_GUARD_FAIL_OPEN='1'
export GATEWAY_OUTPUT_GUARD_TEMPLATE="I'm sorry, I can't help with that."
# 若裁判需 Bearer：再设 GATEWAY_OUTPUT_GUARD_BEARER
# 默认：已配置 URL 时要求请求带 X-Gateway-Output-Guard: 1；仅评测机可保持默认

go run ./cmd/gateway
```

**OpenAI 官方 API 时**将 `GATEWAY_UPSTREAM` 设为 `https://api.openai.com/v1`，并导出 `OPENAI_API_KEY`（或 `GATEWAY_UPSTREAM_API_KEY`），`--openai-model` 与 `curl` 的 `model` 改为例如 `gpt-4o-mini`。

**终端 A — 启裁判（与 `GATEWAY_OUTPUT_GUARD_URL` 同机同端口）：**

```bash
cd Decoupled-LLM-Gateway
JUDGE_BACKEND=heuristic python3 experiments/judge_service/server.py --host 127.0.0.1 --port 8765
# 需要更强判定时可换 openai_moderation / chat_completion（见 README「论文实验数据」）
```

**终端 B — 启网关**：使用上一段 env 后 `go run ./cmd/gateway`（或 `make run-gateway`，但须在命令前 **额外 export** `GATEWAY_OUTPUT_GUARD_*`，因默认 Makefile 未带守卫）。

**终端 C — 最小评测（经网关 + 自动加输出守卫头）**：

`--gateway-output-guard` 会为 **非 `direct_upstream`** 的 defense 附加 `X-Gateway-Output-Guard: 1`。最小场景与 §2.4 一致（不含 `direct_upstream`，避免与「仅经网关才带头」混淆）。

```bash
cd Decoupled-LLM-Gateway
export DEEPSEEK_API_KEY='你的密钥'
mkdir -p results

python3 experiments/run_paper_benchmark.py \
  --gateway-url http://127.0.0.1:8080 \
  --upstream-url https://api.deepseek.com \
  --openai-model deepseek-chat \
  --defenses unified,no_obfuscate,no_decoy,intent_only \
  --scenarios benign_baseline,refusal_keyword,extraction_leak \
  --gateway-output-guard \
  --seed 42 \
  -o results/trackA_core_guard_seed42.json
```

**说明**：`manifest.gateway_output_guard_header` 为 `true`。守卫与启发式裁判可能对 **良性提示** 给出「非拒绝」，从而触发模板覆盖——这是该机制的预期语义；若仅想测有害侧，可缩小 `--scenarios` 或暂时去掉 `benign_baseline`，并确保使用场景符合合规与费用预期。本地链路可先跑 **`make smoke-output-guard`**（§1.5）再换真实密钥。

### 2.9 真实上游小样本对比（`harmful_rsr_suite` + `benign_fpr_suite`，有/无输出守卫头）

在 **§2.8** 已起 **裁判 `8765` + 带 `GATEWAY_OUTPUT_GUARD_URL` 的网关** 的前提下，用同一上游与同一模型做两次评测：**仅差异**为是否加 **`--gateway-output-guard`**。用于核对 `manifest`、粗看 `per_prompt` 与费用可控。

**条数上限**：`--max-harmful-prompts N`、`--max-benign-fpr-prompts M` 只截取提示文件**前 N / M 条**（仍计入 `manifest.max_harmful_prompts` / `max_benign_fpr_prompts`）；**其它场景**（若使用 `--suite full`）不受这两个参数影响，仅 **`harmful_rsr_suite` / `benign_fpr_suite`** 被截断。

**终端 C — 先无守卫头（基线）再带守卫头**（DeepSeek 示例；`UPSTREAM` 与 §2.4 一致）：

```bash
cd Decoupled-LLM-Gateway
export DEEPSEEK_API_KEY='你的密钥'
mkdir -p results

BASE=( python3 experiments/run_paper_benchmark.py
  --gateway-url http://127.0.0.1:8080
  --upstream-url https://api.deepseek.com
  --openai-model deepseek-chat
  --defenses unified,no_obfuscate,no_decoy,intent_only
  --scenarios harmful_rsr_suite,benign_fpr_suite
  --max-harmful-prompts 3
  --max-benign-fpr-prompts 4
  --seed 42
)

"${BASE[@]}" -o results/real_smoke_noguard.json
"${BASE[@]}" --gateway-output-guard -o results/real_smoke_guard.json
```

**核对 manifest**（应仅有 `gateway_output_guard_header` 等预期字段差异）：

```bash
jq '.manifest | {gateway_output_guard_header, max_harmful_prompts, max_benign_fpr_prompts, protocol_version, openai_model_field}' \
  results/real_smoke_noguard.json results/real_smoke_guard.json
diff -u <(jq '.manifest | del(.git_sha, .hostname)' results/real_smoke_noguard.json) \
        <(jq '.manifest | del(.git_sha, .hostname)' results/real_smoke_guard.json) || true
```

**主观观感**：对每条 defense × 场景，查看 `runs[0].results[]` 中 `scenario_id` 为 `harmful_rsr_suite` / `benign_fpr_suite` 的 **`per_prompt`**（`rsr` / `rsr_false_positive`、必要时对照助手正文是否被模板替换）。带守卫时，经网关且模型未形成「拒绝形」回复的完成体可能被 **`GATEWAY_OUTPUT_GUARD_TEMPLATE`** 覆盖，有害集上 RSR 可能**升高**，良性集上不当拒绝率也可能变化——需结合裁判与业务预期解读。

**一键脚本**（与上面等价；需已配置密钥且网关/裁判已就绪）：

```bash
bash experiments/scripts/run_real_upstream_guard_compare.sh
# 可选：MAX_HARMFUL=2 MAX_BENIGN_FPR=3 OPENAI_MODEL=gpt-4o-mini UPSTREAM_URL=https://api.openai.com/v1 ...
```

**`--suite full` + 小样本有害/良性**：在仍要跑全场景但控制这两类费用时，例如：

```bash
python3 experiments/run_paper_benchmark.py --suite full \
  --gateway-url http://127.0.0.1:8080 \
  --upstream-url https://api.deepseek.com \
  --openai-model deepseek-chat \
  --defenses unified,no_obfuscate,no_decoy,intent_only,direct_upstream,smooth_llm \
  --max-harmful-prompts 2 \
  --max-benign-fpr-prompts 3 \
  --gateway-output-guard \
  --seed 42 \
  -o results/trackA_full_capped_guard.json
```

`direct_upstream` **不会**带 `X-Gateway-Output-Guard`，可与经网关防御对照。

---

## 三、两种模式对照

| 项目 | echo-llm | 真实 API |
|------|-----------|-----------|
| 用途 | 烟测、CI、`paper-eval-check`、协议对齐 | 论文主表、延迟与 RSR/F1 **真实**分布 |
| 密钥 | 不需要 | 需要 |
| `--upstream-url` | `http://127.0.0.1:9090` | `https://api.deepseek.com` 等 |
| `--openai-model` | `eval` / `mock` 等占位 | `deepseek-chat`、`gpt-4o-mini` 等真实 id |

---

## 四、更多文档

| 文件 | 内容 |
|------|------|
| `README.md` | 混淆 profile、Redis/M3、实验头、环境变量全表 |
| `experiments/EXPERIMENT_PLAYBOOK_CN.md` | Track A 实验与论文补充：能力表、边界、从零到正式数据、JSON→论文章节 |
| `experiments/PIPELINE_FORMAL_DATA.md` | 论文流水线、正式数据归档、`direct_upstream` Bearer |
| `docs/DEFENSE_INVENTORY_AND_LITERATURE.md` | 已实现防御与文献对照 |

---

## 五、常见问题

**网关 502，且错误里出现 `127.0.0.1:9090`、`connection refused`**  
经 `unified` 等防御时，流量走网关；网关默认上游为 **echo `9090`**。请 **`export GATEWAY_UPSTREAM=https://api.deepseek.com`（或你的 API）并设置密钥后重启网关**；或启动 `make run-echo`。勿误以为评测脚本里的 **`--upstream-url` 会改变网关上游**——它只影响 **`direct_upstream`** 直连。

**网关 502 / 其它 upstream error**  
检查 `GATEWAY_UPSTREAM` 是否带协议与主机；DeepSeek 一般为 `https://api.deepseek.com`（无尾部 `/v1` 路径重复问题由网关拼接）。

**direct_upstream 报 401**  
导出 `DEEPSEEK_API_KEY` 或 `GATEWAY_UPSTREAM_API_KEY`；脚本直连上游时会自动加 Bearer。

**jq 查看 JSON**  
过滤器在前、文件在后：`jq '.runs[0].results[0]' results/trackA_core_seed42.json`。
