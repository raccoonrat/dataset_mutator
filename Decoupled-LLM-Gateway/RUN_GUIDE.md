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

### 1.4 仅测脚本逻辑（不启 echo/网关）

```bash
make paper-eval-check
# 等价：python3 experiments/run_paper_benchmark.py --self-check
```

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

日志中应出现上游认证已开启（**不会**打印密钥）。若需同步日志顺序调试，可设 `GATEWAY_LOG_AFTER_RESPONSE=0`（见 `README.md` 环境变量表）。

### 2.3 快速验证（经网关）

```bash
curl -sS http://127.0.0.1:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"deepseek-chat","messages":[{"role":"user","content":"用一句话说你好"}]}'
```

### 2.4 跑论文评测脚本（经网关 + 真实模型）

`--openai-model` 写入请求体与 JSON `manifest`，须与服务商模型 id 一致（如 `deepseek-chat`）。

**`--upstream-url`** 必须与网关相同（供 `direct_upstream` 直连对比）；脚本会从环境变量读 Bearer。

```bash
export DEEPSEEK_API_KEY='你的密钥'
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
| `experiments/PIPELINE_FORMAL_DATA.md` | 论文流水线、正式数据归档、`direct_upstream` Bearer |
| `docs/DEFENSE_INVENTORY_AND_LITERATURE.md` | 已实现防御与文献对照 |

---

## 五、常见问题

**网关 502 / upstream error**  
检查 `GATEWAY_UPSTREAM` 是否带协议与主机；DeepSeek 一般为 `https://api.deepseek.com`（无尾部 `/v1` 路径重复问题由网关拼接）。

**direct_upstream 报 401**  
导出 `DEEPSEEK_API_KEY` 或 `GATEWAY_UPSTREAM_API_KEY`；脚本直连上游时会自动加 Bearer。

**jq 查看 JSON**  
过滤器在前、文件在后：`jq '.runs[0].results[0]' results/trackA_core_seed42.json`。
