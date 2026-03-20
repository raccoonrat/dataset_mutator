# Decoupled-LLM-Gateway

**Milestone 1（当前）**：可在普通开发者机器上跑通的最小可行架构（MVP）——同步网关 + Mock 大模型 + 清晰契约 + 可 CI 验证的测试与流水线。  
目标不是堆功能，而是 **边界清楚、数据结构克制、热路径保持愚蠢（dumb）**，为后续「异步物理隔离」的治理环留出稳定插口。

---

## Milestone 1 交付范围

| 交付项 | 状态 |
|--------|------|
| Go 同步网关，OpenAI 兼容入口 `POST /v1/chat/completions` | ✅ |
| 不接真实 LLM：`echo-llm` 模拟上游 | ✅ |
| 上下文混淆（MVP 规则）+ 系统侧诱饵注入 | ✅ |
| 进程内策略缓存 + 命中则模板降级（不调上游） | ✅ |
| 网关 → 异步环：每请求一行 NDJSON（`GatewayLogEvent`） | ✅ |
| Python Worker 桩（stdin 消费日志） | ✅ |
| 单元/集成测试 + GitHub Actions | ✅ |

**M1 验收口径**：`Request → [策略快照匹配] → [上下文混淆 + 诱饵] → [Mock LLM] → Response` 全链路可复现；`make test` 通过；可选策略种子下第二次同类请求可走降级路径（与 M3 闭环演示衔接）。

---

## 设计哲学（与「同步 / 异步解耦」一致）

1. **极简关键路径（Keep the Hot Path Dumb）**  
   同步网关只做轻量字符串处理、JSON 中 `messages` 的改写与 HTTP 转发，**不在返回前等待**任何重推理或外部裁判。

2. **异步容错（Fail Async）**  
   复杂判定（混合裁判、元认知等）归属下半部；本里程碑仅提供 **日志契约与 Worker 占位**，不把重逻辑塞进网关。

3. **优雅降级优先（Degrade Gracefully）**  
   命中本地策略（或后续扩展的硬熔断）时，**立即**返回预设模板响应，避免上游挂起拖死业务。

---

## 系统架构：上半部与下半部

### 上半部：同步网关（低延迟）

- **实现**：`cmd/gateway`，核心逻辑在 `internal/gateway`。
- **职责（M1）**  
  - **上下文混淆**：对用户相关文本做 ID 类剥离（`internal/obfuscate`）。  
  - **诱饵注入**：在系统消息中注入可追踪 token（`internal/decoy` + `internal/chat`）。  
  - **拦截与降级**：读取进程内策略（`internal/policy`）；匹配则返回模板 JSON，否则转发上游。  
- **延迟预期**：同步侧仅为正则/JSON 与一次 HTTP 客户端调用；不设重计算（具体数值依赖机器与网络，设计目标与论文中「毫秒级附加」一致，由后续基准补充）。

### 下半部：异步治理环（占位）

- **实现**：`worker/main.py`（M1 仅从 stdin 读 NDJSON）。  
- **后续（M3）**：消费队列（如 Redis Streams / NATS）、混合裁判、将 `PolicyRule` 写入网关可读缓存。

---

## 核心数据契约（API 边界）

网关与异步环 **只通过约定字段通信**，避免隐式耦合。

### 1. 网关 → 异步环（日志推流）

每处理完一次请求，向配置的 Sink 输出 **一行 JSON**（默认 stdout）：

```json
{
  "trace_id": "req-1a2b3c",
  "timestamp": 1710892800,
  "injected_decoy_id": "decoy-xxxxxxxx",
  "raw_user_prompt": "原始用户侧拼接快照",
  "obfuscated_prompt": "混淆后的用户侧快照",
  "llm_response": "上游返回摘要或全文片段",
  "degradation_triggered": false
}
```

### 2. 异步环 → 网关（策略缓存，M1 为文件种子 / 内存）

```json
{
  "rule_id": "rule-malicious-compliance-01",
  "action": "DEGRADE_TO_TEMPLATE",
  "trigger_pattern": "Go 正则表达式，匹配 obfuscated_prompt",
  "template_response": "抱歉，该请求违反安全策略，已被拒绝。"
}
```

启动时可通过 `GATEWAY_POLICY_SEED_FILE` 加载 JSON **数组**；运行时由 Worker 写入等价结构（M3 接 Redis 等）。

---

## 仓库布局

| 路径 | 说明 |
|------|------|
| `cmd/gateway` | 同步网关入口 |
| `cmd/echo-llm` | Mock LLM（回声 OpenAI 形响应） |
| `internal/contracts` | 上述 JSON 的 Go 结构体 |
| `internal/chat` | 仅改写 `messages`，保留其余请求字段 |
| `internal/obfuscate` | 混淆规则（M2 可扩展） |
| `internal/decoy` | 诱饵 ID 生成与注入 |
| `internal/policy` | `Store` 接口与内存实现 |
| `internal/gateway` | HTTP 热路径编排 |
| `internal/logsink` | NDJSON 输出 |
| `worker/main.py` | 异步环桩 |
| `examples/policy_seed.json` | 演示用策略种子 |

---

## 快速开始

**终端 1 — Mock LLM**

```bash
cd Decoupled-LLM-Gateway
make run-echo
```

**终端 2 — 网关**（默认上游 `http://127.0.0.1:9090`）

```bash
cd Decoupled-LLM-Gateway
make run-gateway
```

**一键脚本**（后台起 echo，前台起 gateway）

```bash
bash scripts/dev.sh
```

**调用示例**（`temperature` 等字段会原样转发，仅 `messages` 被网关处理）

```bash
curl -sS http://127.0.0.1:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"mock","temperature":0.7,"messages":[{"role":"user","content":"hello"}]}'
```

**将日志管道给 Worker 桩**（网关访问日志在 stdout，运行日志在 stderr）

```bash
make run-gateway 2>/dev/null | python3 worker/main.py
```

---

## 环境变量

| 变量 | 默认 | 含义 |
|------|------|------|
| `GATEWAY_LISTEN` | `:8080` | 网关监听地址 |
| `GATEWAY_UPSTREAM` | `http://127.0.0.1:9090` | 上游 OpenAI 兼容 API 根 URL |
| `GATEWAY_UPSTREAM_TIMEOUT_MS` | `30000` | 上游 HTTP 超时 |
| `GATEWAY_MAX_BODY_BYTES` | `4194304` | 请求/响应体大小上限 |
| `GATEWAY_ASYNC_LOG` | `1` | `0` 关闭 NDJSON 事件输出 |
| `GATEWAY_POLICY_SEED_FILE` | 空 | 启动时加载策略 JSON 数组 |
| `ECHO_LISTEN` | `:9090` | 回声服务监听地址 |

---

## 策略种子与防御性降级

演示文件：`examples/policy_seed.json`。

```bash
GATEWAY_POLICY_SEED_FILE=examples/policy_seed.json go run ./cmd/gateway
```

当 **混淆后的用户侧快照** `obfuscated_prompt` 对应的拼接文本匹配某条规则的 `trigger_pattern`，且 `action` 为 `DEGRADE_TO_TEMPLATE` 时，网关 **不调用上游**，直接返回 OpenAI 形的完成 JSON，正文为 `template_response`。

---

## 开发与 CI

```bash
make test    # go test ./... -race -count=1
make vet     # go vet ./...
```

仓库根目录下 GitHub Actions：`.github/workflows/decoupled-llm-gateway.yml`（变更 `Decoupled-LLM-Gateway/**` 时触发 Go 测试/vet 与 Worker stdin 冒烟）。

---

## 路线图

| 里程碑 | 内容 |
|--------|------|
| **M1（当前）** | 骨架 + Echo + 混淆/诱饵/策略降级 + NDJSON + Worker 桩 + CI |
| **M2** | 强化上下文混淆引擎（ID 对齐、更多单测与基准） |
| **M3** | 队列、混合裁判、策略下发（如 Redis）与网关缓存刷新，完成「诱饵异常 → 规则 → 第二次请求前置拦截」闭环 |

---

## 许可

与父仓库一致；若单独拆库，请在本目录补充 `LICENSE` 并视情况修改 `go.mod` 的 `module` 路径。
